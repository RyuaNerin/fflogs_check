package analysispool

import (
	"fmt"
	"hash/fnv"
	"sort"
	"time"
	"unicode"
	"unicode/utf8"

	"ffxiv_check/analysis"
	"ffxiv_check/cache"
	"ffxiv_check/ffxiv"
)

var (
	csStatistics = cache.NewStorage("./_cachedata/statistics", time.Hour, "./analysis", "./ffxiv")
)

func getOptionHash(ao *analysis.AnalyzeOptions) uint64 {
	h := fnv.New64()

	b := make([]byte, 8)
	append := func(s string) {
		for _, c := range s {
			r := unicode.ToLower(c)

			if r >= 0 {
				if r < utf8.RuneSelf {
					b[0] = byte(r)
					h.Write(b[:1])
				} else {
					n := utf8.EncodeRune(b, r)
					h.Write(b[:n])
				}
			}
		}
	}

	var ss ffxiv.SkillSets
	if ao.CharRegion == "kr" {
		ss = ffxiv.Korea
	} else {
		ss = ffxiv.Global
	}

	sort.Ints(ao.Encouters)
	sort.Ints(ao.AdditionalPartitions)
	sort.Strings(ao.Jobs)

	fmt.Fprint(
		h,
		ss.Hash, "|",
	)
	append(ao.CharName)
	append(ao.CharServer)
	append(ao.CharRegion)

	fmt.Fprint(
		h,
		ao.Encouters, "|",
		ao.AdditionalPartitions, "|",
	)
	for _, jobs := range ao.Jobs {
		append(jobs)
	}

	return h.Sum64()
}
