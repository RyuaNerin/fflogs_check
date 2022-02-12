package analysis

import (
	"log"
	"math"

	"ffxiv_check/ffxiv"
)

// avg, med 계산을 위해 사용 횟수 등을 체크하는 부분
func (inst *analysisInstance) buildReportCaclPrepare(stat *Statistic) {
	getEncData := func(encounterID int) *StatisticEncounter {
		encData, ok := stat.encountersMap[encounterID]
		if !ok {
			encData = &StatisticEncounter{
				ID:      encounterID,
				Name:    inst.encounterNames[encounterID],
				jobsMap: make(map[string]*StatisticEncounterJob, len(inst.InpCharJobs)),
			}
			stat.encountersMap[encounterID] = encData
		}

		return encData
	}
	getEncJobData := func(encData *StatisticEncounter, job string) *StatisticEncounterJob {
		encJobData, ok := encData.jobsMap[job]
		if !ok {
			encJobData = &StatisticEncounterJob{
				ID:        ffxiv.JobOrder[job],
				Job:       job,
				skillsMap: make(map[int]*StatisticSkill, len(inst.skillSets.Job[job])),
			}
			encData.jobsMap[job] = encJobData
		}

		return encJobData
	}
	getBuffUsage := func(encJob *StatisticEncounterJob, skillInfo ffxiv.SkillData) *StatisticSkill {
		buffUsage, ok := encJob.skillsMap[skillInfo.ID]
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
			encJob.skillsMap[skillInfo.ID] = buffUsage
		}
		return buffUsage
	}

	for _, fightData := range inst.Fights {
		if !fightData.DoneEvents {
			continue
		}

		encCur := getEncData(fightData.EncounterID)
		encCur.Kills++
		encCurJob := getEncJobData(encCur, fightData.Job)
		encCurJob.Kills++

		encAll := getEncData(0)
		encAll.Kills++
		encAllJob := getEncJobData(encAll, fightData.Job)
		encAllJob.Kills++

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

			encCurJobSkill := getBuffUsage(encCurJob, skillInfo)
			encAllJobSkill := getBuffUsage(encAllJob, skillInfo)

			fightSkillData := fightData.skillData[skillId]

			encCurJobSkill.Usage.data = append(encCurJobSkill.Usage.data, fightSkillData.Used)
			encAllJobSkill.Usage.data = append(encAllJobSkill.Usage.data, fightSkillData.Used)

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

				encCurJobSkill.Cooldown.data = append(encCurJobSkill.Cooldown.data, float32(cooldown))
				encAllJobSkill.Cooldown.data = append(encAllJobSkill.Cooldown.data, float32(cooldown))

				if skillInfo.ContainsInScore {
					switch skillId {
					case ffxiv.SkillIdReduceDamangeDebuff:
						cooldown = 1 - float64(fightSkillData.UsedForPercent)/float64(fightSkillData.MaxForPercent)*100
					}

					jobScore.scoreSum += cooldown
					jobScore.scoreCount++

					jobScoreAll.scoreSum += cooldown
					jobScoreAll.scoreCount++

					encCur.scoreSum += cooldown
					encCur.scoreCount++

					encCurJob.scoreSum += cooldown
					encCurJob.scoreCount++
				}
			}
		}
	}
}
