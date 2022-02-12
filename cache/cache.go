package cache

import (
	"compress/gzip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

type Storage struct {
	baseDir string

	savingMapLock sync.RWMutex
	savingMap     map[uint64]struct{}

	cleanUpLock sync.RWMutex
}

func NewStorage(dir string, expires time.Duration, hashDir ...string) *Storage {
	if len(hashDir) == 0 {
		panic("empty hashDir")
	}

	s := &Storage{
		baseDir:   dir,
		savingMap: make(map[uint64]struct{}, 16),
	}

	cleanUpWithHash(dir, hashDir...)

	if expires != 0 {
		go s.cleanup(expires)
	}

	return s
}

func (s *Storage) cleanup(expires time.Duration) {
	t := time.NewTicker(30 * time.Second)

	for {
		<-t.C

		t := time.Now().Add(-expires)

		s.cleanUpLock.Lock()
		filepath.Walk(
			s.baseDir,
			func(path string, info fs.FileInfo, err error) error {
				if info == nil {
					return os.ErrInvalid
				}

				if info.IsDir() {
					return nil
				}

				modTime := info.ModTime()
				if modTime.Before(t) {
					err := os.Remove(path)
					if err != nil {
						fmt.Printf("%+v\n", errors.WithStack(err))
						sentry.CaptureException(err)
					}
				}

				return nil
			},
		)
		s.cleanUpLock.Unlock()
	}
}

func (s *Storage) lock(h uint64) bool {
	s.savingMapLock.Lock()
	defer s.savingMapLock.Unlock()

	_, ok := s.savingMap[h]
	if !ok {
		s.savingMap[h] = struct{}{}
	}
	return !ok
}
func (s *Storage) unlock(h uint64) {
	s.savingMapLock.Lock()
	defer s.savingMapLock.Unlock()

	delete(s.savingMap, h)
}
func (s *Storage) check(h uint64) bool {
	s.savingMapLock.RLock()
	defer s.savingMapLock.RUnlock()

	_, ok := s.savingMap[h]
	return ok
}

func (s *Storage) path(h uint64) string {
	return fmt.Sprintf(
		"%s/%02x/%04x/%016x.json.gz",
		s.baseDir,
		(h>>(8*7))&0xFF,
		(h>>(8*6))&0xFFFF,
		h,
	)
}

func (s *Storage) Save(h uint64, r interface{}) {
	s.cleanUpLock.RLock()
	defer s.cleanUpLock.RUnlock()

	if !s.lock(h) {
		return
	}
	defer s.unlock(h)

	fsPath := s.path(h)
	os.MkdirAll(filepath.Dir(fsPath), 0700)

	fs, err := os.Create(fsPath)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return
	}
	defer fs.Close()

	gz := gzip.NewWriter(fs)
	defer gz.Close()

	err = jsoniter.NewEncoder(gz).Encode(r)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		gz.Close()
		fs.Close()
		os.Remove(fsPath)
		return
	}

	err = gz.Flush()
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		gz.Close()
		fs.Close()
		os.Remove(fsPath)
		return
	}
}

func (s *Storage) Load(h uint64, r interface{}) bool {
	s.cleanUpLock.RLock()
	defer s.cleanUpLock.RUnlock()

	if s.check(h) {
		return false
	}

	fsPath := s.path(h)

	fs, err := os.Open(fsPath)
	if err != nil {
		return false
	}
	defer fs.Close()

	gz, err := gzip.NewReader(fs)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return false
	}
	defer gz.Close()

	err = jsoniter.NewDecoder(gz).Decode(r)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return false
	}
	return true
}
