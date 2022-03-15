package allstar

import (
	"fmt"
	"log"

	"ffxiv_check/analysis/allstar/allstardata"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

func (inst *analysisInstance) UpdateGlobalRank() bool {
	inst.progress("[3 / 3] 등수 계산 중")

	log.Printf("UpdateKrEncounterRank %s@%s\n", inst.CharName, inst.CharServer)

	for _, jobData := range inst.tmplData.jobsMap {
		for _, partData := range jobData.partitionsMap {
			var allstarSum float32
			var kills int
			for _, encData := range partData.encountersMap {
				if encData.Rdps > 0 {
					r, err := allstardata.GetEncounterRank(
						inst.ctx,
						encData.EncounterID,
						partData.PartitionIDGlobal,
						jobData.Job,
						encData.Rdps,
					)
					if err != nil {
						sentry.CaptureException(err)
						fmt.Printf("%+v\n", errors.WithStack(err))
						return false
					}
					encData.Global.Rank = r.Rank
					encData.Global.RankPercent = r.RankPercent
					encData.Global.Allstar = r.AllstarPoint

					allstarSum += r.AllstarPoint

					kills += encData.Kills
				}
			}

			// 빠진 항목 채워 넣기
			for encId, encInfo := range inst.Preset.EncounterMap {
				_, ok := partData.encountersMap[encId]
				if ok {
					continue
				}

				partData.encountersMap[encId] = &tmplDataEncounter{
					EncounterID:   encId,
					EncounterName: encInfo.Name,
					Rdps:          -1,
				}
			}

			if inst.Preset.UseAllstarRank {
				r, err := allstardata.GetAllstarRank(
					inst.ctx,
					inst.Preset.Zone,
					partData.PartitionIDGlobal,
					jobData.Job,
					allstarSum,
				)
				if err != nil {
					sentry.CaptureException(err)
					fmt.Printf("%+v\n", errors.WithStack(err))
					return false
				}
				partData.Global.Allstar = allstarSum
				partData.Global.Rank = r.Rank
				partData.Global.RankPercent = r.RankPercent
				partData.TotalKills = kills

				if jobData.BestGlobal.Rank == 0 || (jobData.BestGlobal.Rank != -1 && jobData.BestGlobal.Rank > partData.Global.Rank) {
					jobData.BestGlobal = partData.Global
				}
			}
		}
	}

	return true
}
