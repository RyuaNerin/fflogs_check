package analysis

import (
	"log"
	"math"

	"ffxiv_check/ffxiv"
)

// avg, med 계산을 위해 사용 횟수 등을 체크하는 부분
func (inst *analysisInstance) buildReportCaclPrepare(stat *Statistic) {
	for _, fightData := range inst.Fights {
		if !fightData.DoneEvents {
			continue
		}

		encData, ok := stat.encountersMap[fightData.EncounterID]
		if !ok {
			encData = &StatisticEncounter{
				ID:      fightData.EncounterID,
				Name:    inst.encounterNames[fightData.EncounterID],
				jobsMap: make(map[string]*StatisticEncounterJob, len(inst.InpCharJobs)),
			}
			stat.encountersMap[fightData.EncounterID] = encData
		}
		encData.Kills++

		encJobData, ok := encData.jobsMap[fightData.Job]
		if !ok {
			encJobData = &StatisticEncounterJob{
				ID:        ffxiv.JobOrder[fightData.Job],
				Job:       fightData.Job,
				skillsMap: make(map[int]*StatisticSkill, len(inst.skillSets.Job[fightData.Job])),
			}
			encData.jobsMap[fightData.Job] = encJobData
		}
		encJobData.Kills++

		jobScoreAll := stat.jobsMap[""]
		jobScoreAll.Kills++

		jobScore, ok := stat.jobsMap[fightData.Job]
		if !ok {
			jobScore = &StatisticJob{
				ID:  ffxiv.JobOrder[fightData.Job],
				Job: fightData.Job,
			}
			stat.jobsMap[fightData.Job] = jobScore
		}
		jobScore.Kills++

		for _, skillId := range inst.skillSets.Job[fightData.Job] {
			skillInfo := inst.skillSets.Action[skillId]

			buffUsage, ok := encJobData.skillsMap[skillId]
			if !ok {
				buffUsage = &StatisticSkill{
					Info: BuffSkillInfo{
						ID:              skillInfo.ID,
						Cooldown:        skillInfo.Cooldown,
						Name:            skillInfo.Name,
						Icon:            skillInfo.IconUrl,
						WithDowntime:    skillInfo.WithDowntime,
						ContainsInScore: skillInfo.ContainsInScore,
					},
				}
				encJobData.skillsMap[skillId] = buffUsage
			}

			fightSkillData := fightData.skillData[skillId]

			buffUsage.Usage.data = append(buffUsage.Usage.data, fightSkillData.Used)

			// 최대 사용 가능 횟수 세기
			if skillInfo.WithDowntime {
				//cooldown := float64(totalCooldown) / float64(fightTime) * 100
				var cooldown float64 = 0
				if fightSkillData.MaxForPercent > 0 {
					cooldown = float64(fightSkillData.UsedForPercent) / float64(fightSkillData.MaxForPercent) * 100
				}

				if math.IsNaN(cooldown) {
					log.Println("fffffffffffffffff")
				}

				buffUsage.Cooldown.data = append(buffUsage.Cooldown.data, float32(cooldown))

				if skillInfo.ContainsInScore {
					switch skillId {
					case ffxiv.SkillIdReduceDamangeDebuff:
						cooldown = 1 - float64(fightSkillData.UsedForPercent)/float64(fightSkillData.MaxForPercent)*100
					}

					jobScore.scoreSum += cooldown
					jobScore.scoreCount++

					jobScoreAll.scoreSum += cooldown
					jobScoreAll.scoreCount++

					encData.scoreSum += cooldown
					encData.scoreCount++

					encJobData.scoreSum += cooldown
					encJobData.scoreCount++
				}
			}
		}
	}
}
