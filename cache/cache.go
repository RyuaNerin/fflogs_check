package cache

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

type cacheKey struct {
	v0 uint64
	v8 uint64
}

type Storage struct {
	baseDir string

	savingMapLock sync.RWMutex
	savingMap     map[cacheKey]struct{}

	cleanUpLock sync.RWMutex
}

func NewStorage(dir string, expires time.Duration) *Storage {
	s := &Storage{
		baseDir:   dir,
		savingMap: make(map[cacheKey]struct{}, 16),
	}

	if expires != 0 {
		go s.cleanup(expires)
	}

	return s
}

func (s *Storage) cleanup(expires time.Duration) {
	t := time.NewTicker(time.Second)

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

func (s *Storage) getJsonCacheKey(h hash.Hash) (ck cacheKey) {
	hb := h.Sum(nil)

	ck.v0 = binary.BigEndian.Uint64(hb[0:])
	ck.v8 = binary.BigEndian.Uint64(hb[8:])

	return
}

func (s *Storage) lock(hk cacheKey) bool {
	s.savingMapLock.Lock()
	defer s.savingMapLock.Unlock()

	_, ok := s.savingMap[hk]
	if !ok {
		s.savingMap[hk] = struct{}{}
	}
	return !ok
}
func (s *Storage) unlock(hk cacheKey) {
	s.savingMapLock.Lock()
	defer s.savingMapLock.Unlock()

	delete(s.savingMap, hk)
}
func (s *Storage) check(hk cacheKey) bool {
	s.savingMapLock.RLock()
	defer s.savingMapLock.RUnlock()

	_, ok := s.savingMap[hk]
	return ok
}

func (s *Storage) path(h cacheKey) string {
	return fmt.Sprintf(
		"%s/%02x/%016x/%016x-%016x.json.gz",
		s.baseDir,
		(h.v0>>(8*7))&0xFF,
		h.v0,
		h.v0, h.v8,
	)
}

func (s *Storage) Save(h hash.Hash, r interface{}) {
	s.cleanUpLock.RLock()
	defer s.cleanUpLock.RUnlock()

	hk := s.getJsonCacheKey(h)
	if !s.lock(hk) {
		return
	}
	defer s.unlock(hk)

	fsPath := s.path(hk)
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

func (s *Storage) Load(h hash.Hash, r interface{}) bool {
	s.cleanUpLock.RLock()
	defer s.cleanUpLock.RUnlock()

	hk := s.getJsonCacheKey(h)
	if s.check(hk) {
		return false
	}

	fsPath := s.path(hk)

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
