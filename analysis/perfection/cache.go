package perfection

import (
	"fmt"
	"hash/fnv"

	"ffxiv_check/cache"
)

var (
	csReport = cache.NewStorage(
		"./_cachedata/report",
		0,
		"./analysis/perfection/fflogs.go",
		"./analysis/perfection/fflogs_2_fight.go",
		"./analysis/perfection/query/tmplReportSummary.tmpl",
	)
	csEvents = cache.NewStorage(
		"./_cachedata/events",
		0,
		"./analysis/perfection/fflogs.go",
		"./analysis/perfection/fflogs_3_event.go",
		"./analysis/perfection/query/tmplReportCastsEvents.tmpl",
	)
)

func cacheReport(reportId string, fightIds string, r interface{}, saveMode bool) bool {
	h := fnv.New64()
	fmt.Fprintf(
		h,
		"%s_fid_%s",
		reportId, fightIds,
	)
	hi := h.Sum64()

	if saveMode {
		csReport.Save(hi, r)
		return true
	} else {
		return csReport.Load(hi, r)
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
	h := fnv.New64()
	fmt.Fprintf(
		h,
		"%s_fid_%d_sid_%d___est_%d_eet_%d___bst_%d_bet_%d___dst_%d_det_%d",
		reportId, fightId, sourceId,
		eventsStartTime, eventsEndTime,
		buffsStartTime, buffsEndTime,
		deathsStartTime, deathsEndTime,
	)
	hi := h.Sum64()

	if saveMode {
		csEvents.Save(hi, r)
		return true
	} else {
		return csEvents.Load(hi, r)
	}
}
