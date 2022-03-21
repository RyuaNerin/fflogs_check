package perfection

import (
	"ffxiv_check/ffxiv"
)

// avg, med 계산을 위해 사용 횟수 등을 체크하는 부분
func (inst *analysisInstance) buildReportCaclPrepare() {
	getEncData := func(encounterID int) *statisticEncounter {
		encData, ok := inst.stat.encountersMap[encounterID]
		if !ok {
			encData = &statisticEncounter{
				ID:      encounterID,
				Name:    inst.encounterNames[encounterID],
				jobsMap: make(map[string]*statisticEncounterJob, len(ffxiv.JobOrder)),
			}
			inst.stat.encountersMap[encounterID] = encData
		}

		return encData
	}
	getEncJobData := func(encData *statisticEncounter, job string) *statisticEncounterJob {
		encJobData, ok := encData.jobsMap[job]
		if !ok {
			encJobData = &statisticEncounterJob{
				ID:        ffxiv.JobOrder[job],
				Job:       job,
				skillsMap: make(map[int]*statisticSkill, len(inst.skillSets.Job[job])),
			}
			encData.jobsMap[job] = encJobData
		}

		return encJobData
	}
	getBuffUsage := func(encJob *statisticEncounterJob, skillInfo ffxiv.SkillData) *statisticSkill {
		buffUsage, ok := encJob.skillsMap[skillInfo.ID]
		if !ok {
			buffUsage = &statisticSkill{
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

		jobScoreAll := inst.stat.jobsMap[""]
		jobScoreAll.Kills++

		jobScore, ok := inst.stat.jobsMap[fightData.Job]
		if !ok {
			jobScore = &statisticJob{
				ID:  ffxiv.JobOrder[fightData.Job],
				Job: fightData.Job,
			}
			inst.stat.jobsMap[fightData.Job] = jobScore
		}
		jobScore.Kills++

		for _, skillId := range inst.skillSets.Job[fightData.Job] {
			skillInfo := inst.skillSets.Action[skillId]

			encCurJobSkill := getBuffUsage(encCurJob, skillInfo)
			encAllJobSkill := getBuffUsage(encAllJob, skillInfo)

			fightSkillData, ok := fightData.skillData[skillId]
			if !ok {
				continue
			}

			encCurJobSkill.Usage.data = append(encCurJobSkill.Usage.data, fightSkillData.Used)
			encAllJobSkill.Usage.data = append(encAllJobSkill.Usage.data, fightSkillData.Used)

			// 최대 사용 가능 횟수 세기
			if skillInfo.WithDowntime {
				//cooldown := float64(totalCooldown) / float64(fightTime) * 100
				var cooldown float64 = 0
				if fightSkillData.MaxForPercent > 0 {
					cooldown = float64(fightSkillData.UsedForPercent) / float64(fightSkillData.MaxForPercent) * 100
					if cooldown > 100 {
						cooldown = 100
					}
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
