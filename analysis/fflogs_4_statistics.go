package analysis

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"ffxiv_check/ffxiv"

	"github.com/getsentry/sentry-go"
)

func (inst *analysisInstance) buildReport() (stat *Statistic) {
	log.Printf("buildReport %s@%s\n", inst.InpCharName, inst.InpCharServer)

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

	////////////////////////////////////////////////////////////////////////////////////////////////////
	// fflogs -> stat 누적
	for _, fflogsFight := range inst.Fights {
		if !fflogsFight.DoneEvents {
			continue
		}

		encData, ok := stat.encountersMap[fflogsFight.EncounterID]
		if !ok {
			encData = &StatisticEncounter{
				ID:      fflogsFight.EncounterID,
				Name:    inst.encounterNames[fflogsFight.EncounterID],
				jobsMap: make(map[string]*StatisticEncounterJob, len(inst.InpCharJobs)),
			}
			stat.encountersMap[fflogsFight.EncounterID] = encData
		}
		encData.Kills++

		encJobData, ok := encData.jobsMap[fflogsFight.Job]
		if !ok {
			encJobData = &StatisticEncounterJob{
				ID:        ffxiv.JobOrder[fflogsFight.Job],
				Job:       fflogsFight.Job,
				skillsMap: make(map[int]*StatisticSkill, len(inst.skillSets.Job[fflogsFight.Job])),
			}
			encData.jobsMap[fflogsFight.Job] = encJobData
		}
		encJobData.Kills++

		jobScoreAll := stat.jobsMap[""]
		jobScoreAll.Kills++

		jobScore, ok := stat.jobsMap[fflogsFight.Job]
		if !ok {
			jobScore = &StatisticJob{
				ID:  ffxiv.JobOrder[fflogsFight.Job],
				Job: fflogsFight.Job,
			}
			stat.jobsMap[fflogsFight.Job] = jobScore
		}
		jobScore.Kills++

		for _, skillId := range inst.skillSets.Job[fflogsFight.Job] {
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

			fightTime := fflogsFight.EndTime - fflogsFight.StartTime

			used := 0
			nextCooldown := 0
			totalCooldown := 0

			switch skillId {
			case ffxiv.SkillIdDeath:
				used = len(fflogsFight.Deaths)

			case ffxiv.SkillIdPotion:
				for _, event := range fflogsFight.Buffs {
					if event.removed {
						event.timestamp = event.timestamp - ffxiv.PotionBuffTime
					}
					if nextCooldown > 0 && event.timestamp < nextCooldown {
						// 적용 후 꺼진 버프
						// 탕약 버프가 두번 뜨는 경우가 있음
						continue
					}

					used++
					nextCooldown = event.timestamp + skillInfo.Cooldown*1000
					totalCooldown += skillInfo.Cooldown * 1000
				}

			default:
				for _, event := range fflogsFight.Casts {
					if skillId != 0 && event.gameID != skillId {
						continue
					}

					if skillInfo.WithDowntime {
						nextCooldown = event.timestamp + skillInfo.Cooldown*1000
						totalCooldown += skillInfo.Cooldown * 1000
					}

					used++
				}
			}

			if nextCooldown > fightTime {
				totalCooldown -= nextCooldown - fightTime
			}

			cooldown := float64(totalCooldown) / float64(fightTime) * 100

			buffUsage.Usage.data = append(buffUsage.Usage.data, used)
			buffUsage.Cooldown.data = append(buffUsage.Cooldown.data, float32(cooldown))

			if skillInfo.ContainsInScore {
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

	////////////////////////////////////////////////////////////////////////////////////////////////////
	// stat 계산
	for _, jobData := range stat.jobsMap {
		if jobData.scoreCount > 0 {
			jobData.Score = float32(jobData.scoreSum / float64(jobData.scoreCount))
		}
	}

	for _, encData := range stat.encountersMap {
		if encData.scoreCount > 0 {
			encData.Score = float32(encData.scoreSum / float64(encData.scoreCount))
		}

		for _, endJobData := range encData.jobsMap {
			if endJobData.scoreCount > 0 {
				endJobData.Score = float32(endJobData.scoreSum / float64(endJobData.scoreCount))
			}

			for _, encJobSkillData := range endJobData.skillsMap {
				sort.Slice(encJobSkillData.Usage.data, func(i, k int) bool { return encJobSkillData.Usage.data[i] < encJobSkillData.Usage.data[k] })
				sort.Slice(encJobSkillData.Cooldown.data, func(i, k int) bool { return encJobSkillData.Cooldown.data[i] < encJobSkillData.Cooldown.data[k] })

				if len(encJobSkillData.Usage.data) > 0 {
					var usageSum int = 0
					for _, u := range encJobSkillData.Usage.data {
						usageSum += u
					}
					encJobSkillData.Usage.Med = encJobSkillData.Usage.data[len(encJobSkillData.Usage.data)/2]
					encJobSkillData.Usage.Avg = float32(usageSum) / float32(len(encJobSkillData.Usage.data))
				}

				if len(encJobSkillData.Cooldown.data) > 0 {
					var cooldownSum float32 = 0
					for _, u := range encJobSkillData.Cooldown.data {
						cooldownSum += u
					}
					encJobSkillData.Cooldown.Med = encJobSkillData.Cooldown.data[len(encJobSkillData.Cooldown.data)/2]
					encJobSkillData.Cooldown.Avg = cooldownSum / float32(len(encJobSkillData.Cooldown.data))
				}
			}
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////
	// check NaN
	var msgOnce sync.Once
	check := func(v float32) {
		if math.IsNaN(float64(v)) {
			msgOnce.Do(func() {
				sentry.CaptureMessage(fmt.Sprintf(
					"NaN : %s@%s (%s)\nEnc: %+v\nPartition: %+v\nJobs: %+v",
					inst.InpCharName, inst.InpCharServer, inst.InpCharRegion,
					inst.InpEncounterIDs,
					inst.InpAdditionalPartition,
					inst.InpCharJobs,
				))
			})
		}
	}
	for _, jobData := range stat.jobsMap {
		check(jobData.Score)
	}
	for _, encData := range stat.encountersMap {
		check(encData.Score)

		for _, encJobData := range encData.jobsMap {
			check(encJobData.Score)

			for _, encJobSkillData := range encJobData.skillsMap {
				check(encJobSkillData.Usage.Avg)
				check(encJobSkillData.Cooldown.Avg)
				check(encJobSkillData.Cooldown.Med)
			}
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////
	// map to slice
	stat.Jobs = make([]*StatisticJob, 0, len(stat.jobsMap))
	for _, jobData := range stat.jobsMap {
		stat.Jobs = append(stat.Jobs, jobData)
	}
	sort.Slice(
		stat.Jobs,
		func(i, k int) bool {
			return ffxiv.JobOrder[stat.Jobs[i].Job] < ffxiv.JobOrder[stat.Jobs[k].Job]
		},
	)

	stat.Encounters = make([]*StatisticEncounter, 0, len(stat.encountersMap))
	for _, encData := range stat.encountersMap {
		stat.Encounters = append(stat.Encounters, encData)

		encData.Jobs = make([]*StatisticEncounterJob, 0, len(encData.jobsMap))
		for _, encJobData := range encData.jobsMap {
			encData.Jobs = append(encData.Jobs, encJobData)

			encJobData.Skills = make([]*StatisticSkill, 0, len(encJobData.skillsMap))
			for _, encJobSkillData := range encJobData.skillsMap {
				encJobData.Skills = append(encJobData.Skills, encJobSkillData)
			}
			sort.Slice(
				encJobData.Skills,
				func(i, k int) bool {
					return inst.skillSets.Action[encJobData.Skills[i].Info.ID].OrderIndex < inst.skillSets.Action[encJobData.Skills[k].Info.ID].OrderIndex
				},
			)
		}
		sort.Slice(
			encData.Jobs,
			func(i, k int) bool {
				return ffxiv.JobOrder[encData.Jobs[i].Job] < ffxiv.JobOrder[encData.Jobs[k].Job]
			},
		)
	}
	sort.Slice(
		stat.Encounters,
		func(i, k int) bool {
			ir, kr := -1, -1

			for ii, v := range inst.InpEncounterIDs {
				if v == stat.Encounters[i].ID {
					ir = ii
				}
			}
			for ii, v := range inst.InpEncounterIDs {
				if v == stat.Encounters[k].ID {
					kr = ii
				}
			}

			return ir < kr
		},
	)

	return stat
}
