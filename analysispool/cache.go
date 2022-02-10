package analysispool

import (
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ffxiv_check/analysis"
	"ffxiv_check/cache"
	"ffxiv_check/ffxiv"
)

var (
	csStatistics      = cache.NewStorage("./_cachedata/statistics", time.Hour)
	analysisFilesHash uint32
)

func init() {
	h := fnv.New32a()

	read := func(path string) {
		fs, err := os.Open(path)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(h, fs)
		if err != nil && err != io.EOF {
			panic(err)
		}
	}

	var walk func(dir string)

	walk = func(dir string) {
		fiList, err := os.ReadDir(dir)
		if err != nil {
			panic(err)
		}

		for _, fi := range fiList {
			path := filepath.Join(dir, fi.Name())
			fmt.Fprint(h, path)

			if fi.IsDir() {
				walk(path)
			} else {
				read(path)
			}
		}
	}

	walk("./analysis")
	walk("./ffxiv")

	analysisFilesHash = h.Sum32()
}

func checkOptionValidation(ao *analysis.AnalyzeOptions) bool {
	ao.CharName = strings.TrimSpace(ao.CharName)
	ao.CharServer = strings.TrimSpace(ao.CharServer)
	ao.CharRegion = strings.TrimSpace(ao.CharRegion)

	switch {
	case len(ao.CharName) < 3:
	case len(ao.CharName) > 20:
	case len(ao.CharServer) < 3:
	case len(ao.CharServer) > 10:
	case len(ao.CharRegion) < 2:
	case len(ao.CharRegion) > 5:
	case len(ao.Encouters) == 0:
	case len(ao.Encouters) > 10:
	case len(ao.AdditionalPartitions) > 5:
	case len(ao.Jobs) == 0:
	case len(ao.Jobs) > len(ffxiv.JobOrder):
	default:
		return true
	}

	return false
}

func getOptionHash(ao *analysis.AnalyzeOptions) hash.Hash {
	var ss ffxiv.SkillSets
	if ao.CharRegion == "kr" {
		ss = ffxiv.Korea
	} else {
		ss = ffxiv.Global
	}

	sort.Ints(ao.Encouters)
	sort.Ints(ao.AdditionalPartitions)
	sort.Strings(ao.Jobs)

	h := fnv.New128a()
	fmt.Fprint(
		h,
		analysisFilesHash, "|",
		ss.Hash, "|",
		strings.ToLower(ao.CharName), "|",
		strings.ToLower(ao.CharServer), "|",
		strings.ToLower(ao.CharRegion), "|",
		ao.Encouters, "|",
		ao.AdditionalPartitions, "|",
	)
	for _, jobs := range ao.Jobs {
		fmt.Fprintf(h, strings.ToLower(jobs), "|")
	}

	return h
}
