package cache

import (
	"compress/gzip"
	"fmt"
	"hash/fnv"
	"os"
	"sync"

	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
)

func init() {
	os.MkdirAll("./cached-json/report", 0700)
	os.MkdirAll("./cached-json/events", 0700)
}

type cacheKey struct {
	h1 uint64
	h2 uint64
}

var (
	cacheSavingLock sync.RWMutex
	cacheSaving     = make(map[cacheKey]struct{}, 32)
)

func lock(h cacheKey) bool {
	cacheSavingLock.Lock()
	defer cacheSavingLock.Unlock()

	_, ok := cacheSaving[h]
	if !ok {
		cacheSaving[h] = struct{}{}
	}
	return !ok
}
func unlock(h cacheKey) {
	cacheSavingLock.Lock()
	defer cacheSavingLock.Unlock()

	delete(cacheSaving, h)
}
func checkSkip(h cacheKey) bool {
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
	h1 := fnv.New64a()
	fmt.Fprint(h1, dir)
	fmt.Fprintf(h1, path, pathArgs...)

	h2 := fnv.New64()
	fmt.Fprint(h2, dir)
	fmt.Fprintf(h2, path, pathArgs...)

	hash := cacheKey{
		h1: h1.Sum64(),
		h2: h2.Sum64(),
	}

	fsPath := fmt.Sprintf("%s/%016x-%016x.json.gz", dir, hash.h1, hash.h2)

	if saveMode {
		if !lock(hash) {
			return false
		}
		defer unlock(hash)

		fs, err := os.Create(fsPath)
		if err != nil {
			sentry.CaptureException(err)
			return false
		}
		defer fs.Close()

		gz := gzip.NewWriter(fs)
		defer gz.Close()

		err = jsoniter.NewEncoder(gz).Encode(r)
		if err != nil {
			sentry.CaptureException(err)
			gz.Close()
			fs.Close()
			os.Remove(fsPath)
			return false
		}

		err = gz.Flush()
		if err != nil {
			sentry.CaptureException(err)
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
			sentry.CaptureException(err)
			return false
		}
		defer gz.Close()

		err = jsoniter.NewDecoder(gz).Decode(r)
		if err != nil {
			sentry.CaptureException(err)
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
