package analysis

import (
	"context"
	"os"
	"sort"
	"sync"
	"time"

	"ffxiv_check/analysis/oauth"
	"ffxiv_check/ffxiv"

	"github.com/joho/godotenv"
)

var (
	client *oauth.Client
)

func init() {
	godotenv.Load(".env")

	client = oauth.New(
		os.Getenv("FFLOGS_V2_OAUTH2_CLIENT_ID"),
		os.Getenv("FFLOGS_V2_OAUTH2_CLIENT_SECRET"),
	)
}

type AnalyzeOptions struct {
	Context              context.Context `json:"-"`
	CharName             string          `json:"char_name"`
	CharServer           string          `json:"char_server"`
	CharRegion           string          `json:"char_region"`
	Encouters            []int           `json:"encounters"`
	AdditionalPartitions []int           `json:"partitions"`
	Jobs                 []string        `json:"jobs"`

	Progress func(p string) `json:"-"`
}

func Analyze(opt *AnalyzeOptions) (stat *Statistics, ok bool) {
	inst := analysisInstance{
		ctx: opt.Context,

		CharName:     opt.CharName,
		CharServer:   opt.CharServer,
		CharRegion:   opt.CharRegion,
		CharJobs:     make(map[string]bool, len(opt.Jobs)),
		EncounterIDs: make([]int, len(opt.Encouters)),

		Reports: make(map[string]*analysisReport),
		Fights:  make(map[fightKey]*analysisFight),

		encounterNames: make(map[int]string, len(opt.Encouters)),

		progressString: make(chan string),
	}
	defer close(inst.progressString)

	copy(inst.EncounterIDs, opt.Encouters)

	if inst.ctx == nil {
		inst.ctx = context.Background()
	}
	for _, job := range opt.Jobs {
		_, ok := ffxiv.JobOrder[job]
		if ok {
			inst.CharJobs[job] = true
		}
	}

	inst.AdditionalPartition = append(inst.AdditionalPartition, opt.AdditionalPartitions...)

	var w sync.WaitGroup
	ctx, ctxCancel := context.WithCancel(opt.Context)
	defer ctxCancel()

	w.Add(1)
	go func() {
		defer w.Done()

		nextMessage := time.Now()
		for {
			select {
			case <-ctx.Done():
				return

			case s := <-inst.progressString:
				if time.Now().Before(nextMessage) {
					continue
				}

				opt.Progress(s)
				nextMessage = time.Now().Add(200 * time.Millisecond)
			}
		}
	}()

	if !inst.updateReports() {
		return nil, false
	}
	if !inst.updateFights() {
		return nil, false
	}
	if !inst.updateEvents() {
		return nil, false
	}

	stat = inst.buildReport()

	ctxCancel()
	w.Wait()

	return
}

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

			for _, event := range fight.Events {
				if skillId != 0 && event.avilityID != skillId {
					continue
				}
				/**
				if skillId == 0 && (event.avilityType != 1 || event.avilityID < 10000000) {
					continue
				}
				*/

				if buffUsage.Info.Icon == "" {
					buffUsage.Info.Icon = event.icon____
				}

				if skillInfo.Cooldown > 0 {
					nextCooldown = event.timestamp + skillInfo.Cooldown*1000

					totalCooldown += skillInfo.Cooldown * 1000
				}

				used++
			}

			if nextCooldown > fightTime {
				totalCooldown -= nextCooldown - fightTime
			}

			buffUsage.Usage.data = append(buffUsage.Usage.data, float64(used))
			buffUsage.Cooldown.data = append(buffUsage.Cooldown.data, float64(totalCooldown)/float64(fightTime)*100.0)
		}
	}

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
