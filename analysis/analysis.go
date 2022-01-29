package analysis

import (
	"context"
	"os"
	"sort"
	"sync"
	"time"

	"ffxiv_check/ffxiv"

	fflogs "github.com/RyuaNerin/go-fflogs"
	"github.com/joho/godotenv"
)

var (
	client *fflogs.Client
)

func init() {
	godotenv.Load(".env")

	opt := fflogs.ClientOpt{
		ApiKey: os.Getenv("FFLOGS_V1_APIKEY"),
	}

	var err error
	client, err = fflogs.NewClient(&opt)
	if err != nil {
		panic(err)
	}
}

type AnalyzeOptions struct {
	Context              context.Context `json:"-"`
	CharName             string          `json:"char_name"`
	CharServer           string          `json:"char_server"`
	CharRegion           string          `json:"char_region"`
	Encouters            []EncounterInfo `json:"encounters"`
	AdditionalPartitions []int           `json:"partitions"`
	Jobs                 []string        `json:"jobs"`

	Progress func(p string) `json:"-"`
}

type EncounterInfo struct {
	ZoneID      int `json:"zone"`
	EncounterID int `json:"encounter"`
}

func Analyze(opt *AnalyzeOptions) (stat *Statistics, err error) {
	inst := instance{
		inputContext:    opt.Context,
		inputCharName:   opt.CharName,
		inputCharServer: opt.CharServer,
		inputCharRegion: fflogs.Region(opt.CharRegion),
		inputJob:        make(map[string]bool, len(opt.Jobs)),

		encounter:      make([]*encounterData, len(opt.Encouters)),
		encounterNames: make(map[int]string, len(opt.Encouters)),

		progressString: make(chan string),
	}
	defer close(inst.progressString)

	if inst.inputContext == nil {
		inst.inputContext = context.Background()
	}
	for i, enc := range opt.Encouters {
		inst.encounter[i] = &encounterData{
			zoneID:      enc.ZoneID,
			encounterID: enc.EncounterID,
			reports:     make(map[string]*reportData),
		}
	}
	for _, job := range opt.Jobs {
		_, ok := ffxiv.JobOrder[job]
		if ok {
			inst.inputJob[job] = true
		}
	}

	inst.inputAdditionalPartition = append(inst.inputAdditionalPartition, opt.AdditionalPartitions...)

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

	err = inst.updateReports()
	if err == nil {
		err = inst.updateFights()
		if err == nil {
			err = inst.updateEvents()
			if err == nil {
				stat = inst.buildReport()
			}
		}
	}

	ctxCancel()
	w.Wait()

	return
}

func (inst *instance) buildReport() (r *Statistics) {
	r = &Statistics{
		CharName:   inst.inputCharName,
		CharServer: inst.inputCharServer,
		CharRegion: inst.inputCharRegion.String(),

		Encounter: make([]*StatisticEncounter, len(inst.encounter)),
	}

	for i, enc := range inst.encounter {
		encData := &StatisticEncounter{
			Encounter: StatisticEncounterInfo{
				ID:   enc.encounterID,
				Zone: enc.zoneID,
				Name: inst.encounterNames[enc.encounterID],
			},
			Jobs:    make([]*StatisticJob, 0, len(ffxiv.JobOrder)),
			jobsMap: make(map[string]*StatisticJob, len(ffxiv.JobOrder)),
		}
		r.Encounter[i] = encData

		for _, report := range enc.reports {
			for _, fight := range report.fightData {
				job, ok := encData.jobsMap[fight.job]
				if !ok {
					job = &StatisticJob{
						Job:     fight.job,
						Data:    make([]*StatisticSkill, 0, 10),
						dataMap: make(map[int]*StatisticSkill, 10),
					}
					encData.Jobs = append(encData.Jobs, job)
					encData.jobsMap[fight.job] = job
				}
				job.TotalKills++

				fightTime := fight.endTime - fight.startTime

				for _, skillId := range ffxiv.SkillDataEachJob[fight.job] {
					skillInfo := ffxiv.SkillDataMap[skillId]

					buffUsage, ok := job.dataMap[skillId]
					if !ok {
						buffUsage = &StatisticSkill{
							Info: BuffSkillInfo{
								ID:       skillInfo.ID,
								Cooldown: skillInfo.Cooldown,
								Name:     skillInfo.Name,
							},
						}
						job.Data = append(job.Data, buffUsage)
						job.dataMap[skillId] = buffUsage
					}

					used := 0
					nextCooldown := 0
					totalCooldown := 0

					for _, event := range fight.events {
						if skillId != 0 && event.avilityID != skillId {
							continue
						}
						if skillId == 0 && (event.avilityType != 1 || event.avilityID < 10000000) {
							continue
						}

						whenSeconds := event.timestamp - fight.startTime

						if skillInfo.Cooldown > 0 {
							nextCooldown = whenSeconds + skillInfo.Cooldown*1000

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
				return ffxiv.JobOrder[encData.Jobs[i].Job] > ffxiv.JobOrder[encData.Jobs[k].Job]
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
