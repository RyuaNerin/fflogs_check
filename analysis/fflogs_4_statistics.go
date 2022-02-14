package analysis

import (
	"fmt"
	"log"
	"time"
)

func (inst *analysisInstance) buildReport() (stat *Statistic) {
	log.Printf("buildReport %s@%s\n", inst.InpCharName, inst.InpCharServer)

	inst.buildReportFight()
	inst.buildReportFightRecalcMaxUsing()

	stat = &Statistic{
		UpdatedAt: time.Now().Format("2006-01-02 15:04:05"),

		CharName:   inst.InpCharName,
		CharServer: inst.InpCharServer,
		CharRegion: inst.InpCharRegion,

		State: inst.charState,

		jobsMap:       make(map[string]*StatisticJob, len(inst.InpCharJobs)+1),
		encountersMap: make(map[int]*StatisticEncounter, len(inst.InpEncounterIDs)),
	}
	stat.jobsMap[""] = &StatisticJob{
		Job: "All",
	}
	stat.encountersMap[0] = &StatisticEncounter{
		ID:      0,
		Name:    "종합",
		jobsMap: make(map[string]*StatisticEncounterJob, len(inst.InpCharJobs)),
	}

	switch inst.InpCharRegion {
	case "kr":
		stat.FFLogsLink = fmt.Sprintf("https://ko.fflogs.com/character/id/%d", inst.charID)
	default:
		stat.FFLogsLink = fmt.Sprintf("https://www.fflogs.com/character/id/%d", inst.charID)
	}

	// stat 계산
	inst.buildReportCaclPrepare(stat)
	inst.buildReportCalcStat(stat)

	// check NaN
	inst.buildReportCheckNaN(stat)

	// map to slice
	inst.buildReportMapToSlice(stat)

	return stat
}
