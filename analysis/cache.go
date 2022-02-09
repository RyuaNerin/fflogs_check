package analysis

import (
	"fmt"
	"hash/fnv"

	"ffxiv_check/cache"
)

var (
	csReport = cache.NewStorage("./cached/report", 0)
	csEvents = cache.NewStorage("./cached/events", 0)
)

func cacheReport(reportId string, fightIds string, r interface{}, saveMode bool) bool {
	h := fnv.New128a()
	fmt.Fprintf(
		h,
		"%s_fid_%s",
		reportId, fightIds,
	)

	if saveMode {
		csReport.Save(h, r)
		return true
	} else {
		return csReport.Load(h, r)
	}
}

func cacheCastsEvent(
	reportId string,
	fightId int,
	sourceId int,
	eventsStartTime int, eventsEndTime int,
	buffsStartTime int, buffsEndTime int,
	deathsStartTime int, deathsEndTime int,
	r interface{},
	saveMode bool,
) bool {
	h := fnv.New128a()
	fmt.Fprintf(
		h,
		"%s_fid_%d_sid_%d___est_%d_eet_%d___bst_%d_bet_%d___dst_%d_det_%d",
		reportId, fightId, sourceId,
		eventsStartTime, eventsEndTime,
		buffsStartTime, buffsEndTime,
		deathsStartTime, deathsEndTime,
	)

	if saveMode {
		csEvents.Save(h, r)
		return true
	} else {
		return csEvents.Load(h, r)
	}
}
