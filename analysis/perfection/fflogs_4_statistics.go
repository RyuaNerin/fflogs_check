package perfection

import (
	"fmt"
	"log"
	"time"

	"ffxiv_check/ffxiv"
)

func (inst *analysisInstance) buildReport() {
	log.Printf("buildReport %s@%s\n", inst.InpCharName, inst.InpCharServer)

	inst.buildReportFight()
	inst.buildReportFightRecalc()

	inst.stat.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	inst.stat.State = inst.charState
	inst.stat.jobsMap = make(map[string]*statisticJob, len(ffxiv.JobOrder)+1)
	inst.stat.encountersMap = make(map[int]*statisticEncounter, len(inst.InpEncounterIDs))

	inst.stat.jobsMap[""] = &statisticJob{
		Job: "All",
	}
	inst.stat.encountersMap[0] = &statisticEncounter{
		ID:      0,
		Name:    "종합",
		jobsMap: make(map[string]*statisticEncounterJob, len(ffxiv.JobOrder)),
	}

	switch inst.InpCharRegion {
	case "kr":
		inst.stat.FFLogsLink = fmt.Sprintf("https://ko.fflogs.com/character/id/%d", inst.charID)
	default:
		inst.stat.FFLogsLink = fmt.Sprintf("https://www.fflogs.com/character/id/%d", inst.charID)
	}

	// stat 계산
	inst.buildReportCaclPrepare()
	inst.buildReportCalcStat()
	inst.buildReportCalcRank()

	// check NaN
	inst.buildReportCheckNaN()

	// map to slice
	inst.buildReportMapToSlice()
}
