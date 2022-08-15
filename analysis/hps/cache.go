package hps

import (
	"fmt"
	"hash/fnv"

	"ffxiv_check/cache"
)

var (
	csFightRankings = cache.NewStorage(
		"./_cachedata/report-rankings",
		0,
		"./analysis/hps/query/FightRankings.tmpl",
		"./analysis/hps/ql_fight_rankings.go",
	)
)

func cacheFightRankings(reportId string, fightId int, r interface{}, saveMode bool) bool {
	h := fnv.New64()
	fmt.Fprintf(
		h,
		"%s_fid_%d",
		reportId, fightId,
	)
	hi := h.Sum64()

	if saveMode {
		csFightRankings.Save(hi, r)
		return true
	} else {
		return csFightRankings.Load(hi, r)
	}
}
