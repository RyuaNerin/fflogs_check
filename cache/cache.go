package cache

import (
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

	fsPath := fmt.Sprintf("%s/%d.json", dir, hash)

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

		err = jsoniter.NewEncoder(fs).Encode(r)
		if err != nil {
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

		err = jsoniter.NewDecoder(fs).Decode(r)
		if err != nil {
			log.Printf("%+v\n", errors.WithStack(err))
			return false
		}
		return true
	}
}

func Report(reportId string, fightIds string, r interface{}, saveMode bool) bool {
	return cache(r, saveMode, "./cached-json/report", "%s_fid_%s", reportId, fightIds)
}
func CastsEvent(reportId string, fightId int, sourceId int, startTime int, endTime int, r interface{}, saveMode bool) bool {
	return cache(r, saveMode, "./cached-json/events", "%s_fid_%d_sid_%d_st_%d_et_%d", reportId, fightId, sourceId, startTime, endTime)
}
