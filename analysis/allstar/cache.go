package allstar

import (
	"fmt"
	"hash/fnv"

	"ffxiv_check/cache"
)

var (
	csEncounterRank = cache.NewStorage(
		"./_cachedata/allstar_rank",
		0,
		"./analysis/allstar/do_2_encounter_rank.go",
		"./analysis/allstar/query/tmplEncounterRank.tmpl",
	)
)

func cacheEncounterRank(charName string, charServer string, zoneID int, partitionID int, spec string, r interface{}, saveMode bool) bool {
	h := fnv.New64()
	fmt.Fprintf(
		h,
		"%s@%s_____zone_%d__partition_%d___%s",
		charName, charServer,
		zoneID, partitionID, spec,
	)
	hi := h.Sum64()

	if saveMode {
		csEncounterRank.Save(hi, r)
		return true
	} else {
		return csEncounterRank.Load(hi, r)
	}
}
