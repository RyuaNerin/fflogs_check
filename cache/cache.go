package cache

import (
	"ffxiv_check/share"
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

func cache(path string, r interface{}, saveMode bool) bool {
	h := fnv.New64a()
	h.Write(share.S2b(path))
	hash := h.Sum64()

	if saveMode {
		if !lock(hash) {
			return false
		}
		defer unlock(hash)

		fs, err := os.Create(path)
		if err != nil {
			return false
		}
		defer fs.Close()

		err = jsoniter.NewEncoder(fs).Encode(r)
		if err != nil {
			fs.Close()
			os.Remove(path)
			return false
		}
		return true
	} else {
		if checkSkip(hash) {
			return false
		}

		fs, err := os.Open(path)
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

func Report(reportId string, r interface{}, saveMode bool) bool {
	path := fmt.Sprintf("./cached-json/report/%s.json", reportId)
	return cache(path, r, saveMode)
}
func CastsEvent(reportId string, fightId int, sourceId int, startTime int, r interface{}, saveMode bool) bool {
	path := fmt.Sprintf("./cached-json/events/%s_%d_%d_%d.json", reportId, fightId, sourceId, startTime)
	return cache(path, r, saveMode)
}
