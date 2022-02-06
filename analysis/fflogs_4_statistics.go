package analysis

import (
	"sort"

	"ffxiv_check/ffxiv"
)

var (
	skillIdDeath  = -1
	skillIdPotion = -2

	potionBuffTime = 30 * 1000
)

func (inst *analysisInstance) buildReport() (r *Statistics) {
	r = &Statistics{
		CharName:   inst.CharName,
		CharServer: inst.CharServer,
		CharRegion: inst.CharRegion,

		Encounter: make([]*StatisticEncounter, 0, len(inst.EncounterIDs)),
	}

	encounterMap := make(map[int]*StatisticEncounter)

	for _, fight := range inst.Fights {
		encounterData, ok := encounterMap[fight.EncounterID]
		if !ok {
			encounterData = &StatisticEncounter{
				Encounter: StatisticEncounterInfo{
					ID:   fight.EncounterID,
					Name: inst.encounterNames[fight.EncounterID],
				},
				Jobs:    make([]*StatisticJob, 0, len(ffxiv.JobOrder)),
				jobsMap: make(map[string]*StatisticJob, len(ffxiv.JobOrder)),
			}
			encounterMap[fight.EncounterID] = encounterData
			r.Encounter = append(r.Encounter, encounterData)
		}

		jobData, ok := encounterData.jobsMap[fight.Job]
		if !ok {
			jobData = &StatisticJob{
				Job:     fight.Job,
				Data:    make([]*StatisticSkill, 0, 10),
				dataMap: make(map[int]*StatisticSkill, 10),
			}
			encounterData.Jobs = append(encounterData.Jobs, jobData)
			encounterData.jobsMap[fight.Job] = jobData
		}
		jobData.TotalKills++

		for _, skillId := range ffxiv.SkillDataEachJob[fight.Job] {
			skillInfo := ffxiv.SkillDataMap[skillId]

			buffUsage, ok := jobData.dataMap[skillId]
			if !ok {
				buffUsage = &StatisticSkill{
					Info: BuffSkillInfo{
						ID:       skillInfo.ID,
						Cooldown: skillInfo.Cooldown,
						Name:     skillInfo.Name,
						//Icon:     skillInfo.IconUrl,
					},
				}
				jobData.Data = append(jobData.Data, buffUsage)
				jobData.dataMap[skillId] = buffUsage
			}

			fightTime := fight.EndTime - fight.StartTime

			used := 0
			nextCooldown := 0
			totalCooldown := 0

			switch skillId {
			case skillIdDeath:
				used = len(fight.Deaths)

			case skillIdPotion:
				for _, event := range fight.Buffs {
					if event.removed {
						if event.timestamp < nextCooldown {
							// 적용 후 꺼진 버프
							continue
						} else {
							// 버프 적용이 누락된 경우...
							event.timestamp = event.timestamp - potionBuffTime
						}
					}

					used++
					nextCooldown = event.timestamp + skillInfo.Cooldown*1000
					totalCooldown += skillInfo.Cooldown * 1000
				}

			default:
				for _, event := range fight.Casts {
					if skillId != 0 && event.gameID != skillId {
						continue
					}

					if skillInfo.Cooldown > 0 {
						nextCooldown = event.timestamp + skillInfo.Cooldown*1000
						totalCooldown += skillInfo.Cooldown * 1000
					}

					used++
				}
			}

			if nextCooldown > fightTime {
				totalCooldown -= nextCooldown - fightTime
			}

			buffUsage.Usage.data = append(buffUsage.Usage.data, float64(used))
			buffUsage.Cooldown.data = append(buffUsage.Cooldown.data, float64(totalCooldown)/float64(fightTime)*100.0)
		}
	}

	sort.Slice(
		r.Encounter,
		func(i, k int) bool {
			ir, kr := -1, -1

			for ii, v := range inst.EncounterIDs {
				if v == r.Encounter[i].Encounter.ID {
					ir = ii
				}
			}
			for ii, v := range inst.EncounterIDs {
				if v == r.Encounter[k].Encounter.ID {
					kr = ii
				}
			}

			return ir < kr
		},
	)

	for _, encData := range r.Encounter {
		for _, jobData := range encData.Jobs {
			for _, skillData := range jobData.Data {
				sort.Slice(skillData.Usage.data, func(i, k int) bool { return skillData.Usage.data[i] < skillData.Usage.data[k] })
				sort.Slice(skillData.Cooldown.data, func(i, k int) bool { return skillData.Cooldown.data[i] < skillData.Cooldown.data[k] })

				if len(skillData.Usage.data) > 0 {
					var usageSum float64 = 0
					for _, u := range skillData.Usage.data {
						usageSum += u
					}
					skillData.Usage.Med = skillData.Usage.data[len(skillData.Usage.data)/2]
					skillData.Usage.Avg = float64(usageSum) / float64(len(skillData.Usage.data))
				}

				////////////////////////////////////////////////////////////////////////////////////////////////////

				if len(skillData.Cooldown.data) > 0 {
					var cooldownSum float64 = 0
					for _, u := range skillData.Cooldown.data {
						cooldownSum += u
					}
					skillData.Cooldown.Med = skillData.Cooldown.data[len(skillData.Cooldown.data)/2]
					skillData.Cooldown.Avg = cooldownSum / float64(len(skillData.Cooldown.data))
				}

				////////////////////////////////////////////////////////////////////////////////////////////////////

			}
		}

		sort.Slice(
			encData.Jobs,
			func(i, k int) bool {
				//return ffxiv.JobOrder[encData.Jobs[i].Job] > ffxiv.JobOrder[encData.Jobs[k].Job]
				return encData.Jobs[i].TotalKills > encData.Jobs[k].TotalKills
			},
		)
		for _, job := range encData.jobsMap {
			sort.Slice(
				job.Data,
				func(i, k int) bool {
					return job.Data[i].Info.ID > job.Data[k].Info.ID
				},
			)
		}
	}

	return r
}
