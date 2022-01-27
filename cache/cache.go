package cache

import (
	"fmt"
	"log"
	"os"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

func init() {
	os.MkdirAll("./cached-json/report", 0700)
	os.MkdirAll("./cached-json/events", 0700)
}

var (
	cacheSavingLock sync.RWMutex
	cacheSaving     = make(map[string]struct{}, 32)
)

func lock(path string) bool {
	cacheSavingLock.Lock()
	defer cacheSavingLock.Unlock()

	_, ok := cacheSaving[path]
	if !ok {
		cacheSaving[path] = struct{}{}
	}
	return ok
}
func unlock(path string) {
	cacheSavingLock.Lock()
	defer cacheSavingLock.Unlock()

	delete(cacheSaving, path)
}
func checkSkip(path string) bool {
	cacheSavingLock.RLock()
	defer cacheSavingLock.RUnlock()

	_, ok := cacheSaving[path]
	return ok
}

func cache(path string, r interface{}, saveMode bool) bool {
	if saveMode {
		if lock(path) {
			return false
		}
		defer unlock(path)

		fs, err := os.Create(path)
		if err != nil {
			log.Println(err)
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
		if checkSkip(path) {
			return false
		}

		fs, err := os.Open(path)
		if err != nil {
			log.Println(err)
			return false
		}
		defer fs.Close()

		err = jsoniter.NewDecoder(fs).Decode(r)
		if err != nil {
			log.Println(err)
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
