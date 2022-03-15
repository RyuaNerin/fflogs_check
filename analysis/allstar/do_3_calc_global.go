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

	for _, partData := range inst.tmplData.partitionsMap {
		var bestJob string
		var bestJobRank int

		var best2Job string
		var best2JobAllstar float32

		for _, jobData := range partData.jobsMap {
			var allstarSum float32
			var kills int
			for _, encData := range jobData.encountersMap {
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
				_, ok := jobData.encountersMap[encId]
				if ok {
					continue
				}

				jobData.encountersMap[encId] = &tmplDataEncounter{
					EncounterID:   encId,
					EncounterName: encInfo.Name,
					Rdps:          0,
				}
			}

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
			jobData.Global.Allstar = allstarSum
			jobData.Global.Rank = r.Rank
			jobData.Global.RankPercent = r.RankPercent
			jobData.TotalKills = kills

			if bestJobRank == 0 || (r.Rank != allstardata.Over5000 && r.Rank < bestJobRank) {
				bestJobRank = r.Rank
				bestJob = jobData.Job
			}
			if best2JobAllstar < allstarSum {
				best2JobAllstar = allstarSum
				best2Job = jobData.Job
			}
		}

		if bestJob != "" {
			partData.jobsMap[bestJob].Best = true
		} else if best2Job != "" {
			partData.jobsMap[best2Job].Best = true
		}
	}

	return true
}
