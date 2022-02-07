package cache

import (
	"compress/gzip"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

func init() {
	os.MkdirAll("./cached-json/report", 0700)
	os.MkdirAll("./cached-json/events", 0700)
}

var (
	cacheSavingLock sync.RWMutex
	cacheSaving     = make(map[uint64]struct{}, 32)
)

func lock(h uint64) bool {
	cacheSavingLock.Lock()
	defer cacheSavingLock.Unlock()

	_, ok := cacheSaving[h]
	if !ok {
		cacheSaving[h] = struct{}{}
	}
	return !ok
}
func unlock(h uint64) {
	cacheSavingLock.Lock()
	defer cacheSavingLock.Unlock()

	delete(cacheSaving, h)
}
func checkSkip(h uint64) bool {
	cacheSavingLock.RLock()
	defer cacheSavingLock.RUnlock()

	_, ok := cacheSaving[h]
	return ok
}

func cache(
	r interface{},
	saveMode bool,
	dir string,
	path string,
	pathArgs ...interface{},
) bool {
	h := fnv.New64a()
	fmt.Fprint(h, dir)
	fmt.Fprintf(h, path, pathArgs...)
	hash := h.Sum64()

	fsPath := fmt.Sprintf("%s/%016x.json.gz", dir, hash)

	if saveMode {
		if !lock(hash) {
			return false
		}
		defer unlock(hash)

		fs, err := os.Create(fsPath)
		if err != nil {
			return false
		}
		defer fs.Close()

		gz := gzip.NewWriter(fs)
		defer gz.Flush()

		err = jsoniter.NewEncoder(gz).Encode(r)
		if err != nil {
			gz.Close()
			fs.Close()
			os.Remove(fsPath)
			return false
		}

		err = gz.Flush()
		if err != nil {
			gz.Close()
			fs.Close()
			os.Remove(fsPath)
			return false
		}

		return true
	} else {
		if checkSkip(hash) {
			return false
		}

		fs, err := os.Open(fsPath)
		if err != nil {
			return false
		}
		defer fs.Close()

		gz, err := gzip.NewReader(fs)
		if err != nil {
			log.Printf("%+v\n", errors.WithStack(err))
			return false
		}

		err = jsoniter.NewDecoder(gz).Decode(r)
		if err != nil {
			log.Printf("%+v\n", errors.WithStack(err))
			return false
		}
		return true
	}
}

func Report(reportId string, fightIds string, r interface{}, saveMode bool) bool {
	return cache(
		r, saveMode,
		"./cached-json/report",
		"%s_fid_%s",
		reportId, fightIds,
	)
}
func CastsEvent(
	reportId string,
	fightId int,
	sourceId int,
	eventsStartTime int, eventsEndTime int,
	buffsStartTime int, buffsEndTime int,
	deathsStartTime int, deathsEndTime int,
	r interface{},
	saveMode bool,
) bool {
	return cache(
		r,
		saveMode,
		"./cached-json/events",
		"%s_fid_%d_sid_%d___est_%d_eet_%d___bst_%d_bet_%d___dst_%d_det_%d",
		reportId, fightId, sourceId,
		eventsStartTime, eventsEndTime,
		buffsStartTime, buffsEndTime,
		deathsStartTime, deathsEndTime,
	)
}
