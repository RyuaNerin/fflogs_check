package analysispool

import (
	"fmt"
	"hash"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"ffxiv_check/analysis"
	"ffxiv_check/cache"
	"ffxiv_check/ffxiv"
)

var (
	csStatistics = cache.NewStorage("./cached/statistics", time.Second*10)
)

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
	sort.Ints(ao.Encouters)
	sort.Ints(ao.AdditionalPartitions)
	sort.Strings(ao.Jobs)

	h := fnv.New128a()
	fmt.Fprint(
		h,
		strings.ToLower(ao.CharName), "|||",
		strings.ToLower(ao.CharServer), "|||",
		strings.ToLower(ao.CharRegion), "|||",
		ao.Encouters, "|||",
		ao.AdditionalPartitions, "|||",
		ao.Jobs, "|||",
	)

	return h
}
