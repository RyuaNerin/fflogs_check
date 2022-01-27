package analysis

import (
	"context"
	"os"
	"sort"

	"ffxiv_check/ffxiv"
	_ "ffxiv_check/share"

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
	Zone                 int             `json:"zone"`
	EncouterId           int             `json:"encounter_id"`
	AdditionalPartitions []int           `json:"partitions"`

	Progress func(p float32) `json:"-"`
}

func Analyze(opt *AnalyzeOptions) (*Statistics, error) {
	inst := instance{
		inputContext:     opt.Context,
		inputCharName:    opt.CharName,
		inputCharServer:  opt.CharServer,
		inputCharRegion:  fflogs.Region(opt.CharRegion),
		inputZone:        opt.Zone,
		inputEncounterId: opt.EncouterId,
		reports:          make(map[string]*reportData),
	}
	inst.inputAdditionalPartition = append(inst.inputAdditionalPartition, opt.AdditionalPartitions...)

	err := inst.updateReports()
	if err != nil {
		return nil, err
	}

	err = inst.updateFights()
	if err != nil {
		return nil, err
	}

	err = inst.updateEvents()
	if err != nil {
		return nil, err
	}

	return inst.buildReport(), nil
}

func (inst *instance) buildReport() (r *Statistics) {
	r = &Statistics{
		CharName:    inst.inputCharName,
		CharServer:  inst.inputCharServer,
		CharRegion:  inst.inputCharRegion.String(),
		EncounterId: inst.inputEncounterId,
		Jobs:        make([]*BuffUsageWithJob, 0, len(ffxiv.JobOrder)),
		jobsMap:     make(map[string]*BuffUsageWithJob, len(ffxiv.JobOrder)),
	}

	for _, report := range inst.reports {
		for _, fight := range report.fightData {
			job, ok := r.jobsMap[fight.job]
			if !ok {
				job = &BuffUsageWithJob{
					Job:     fight.job,
					Data:    make([]*BuffUsage, 0, 10),
					dataMap: make(map[int]*BuffUsage, 10),
				}
				r.Jobs = append(r.Jobs, job)
				r.jobsMap[fight.job] = job
			}
			job.TotalKills++

			fightTime := fight.endTime - fight.startTime

			for _, skillId := range ffxiv.SkillDataEachJob[fight.job] {
				skillInfo := ffxiv.SkillDataMap[skillId]

				buffUsage, ok := job.dataMap[skillId]
				if !ok {
					buffUsage = &BuffUsage{
						Skill: BuffSkillInfo{
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
					if event.id != skillId {
						continue
					}

					when := event.timestamp - fight.startTime

					if skillInfo.Cooldown > 0 {
						totalCooldown += when - nextCooldown
						nextCooldown = when + skillInfo.Cooldown
					}

					used++
				}

				if nextCooldown > fightTime {
					totalCooldown = fightTime - nextCooldown
				}

				buffUsage.Usage.data = append(buffUsage.Usage.data, float32(used))
				buffUsage.Cooldown.data = append(buffUsage.Cooldown.data, float32(totalCooldown)/float32(fightTime)*100.0)
			}
		}
	}

	for _, d := range r.Jobs {
		for _, buffUsage := range d.Data {
			sort.Slice(buffUsage.Usage.data, func(i, k int) bool { return buffUsage.Usage.data[i] < buffUsage.Usage.data[k] })
			sort.Slice(buffUsage.Cooldown.data, func(i, k int) bool { return buffUsage.Cooldown.data[i] < buffUsage.Cooldown.data[k] })

			var usageSum float32 = 0
			for _, u := range buffUsage.Usage.data {
				usageSum += u
			}
			buffUsage.Usage.Med = buffUsage.Usage.data[len(buffUsage.Usage.data)/2]
			buffUsage.Usage.Avg = float32(usageSum) / float32(len(buffUsage.Usage.data))

			////////////////////////////////////////////////////////////////////////////////////////////////////

			var cooldownSum float32 = 0
			for _, u := range buffUsage.Cooldown.data {
				cooldownSum += u
			}
			buffUsage.Cooldown.Med = buffUsage.Cooldown.data[len(buffUsage.Cooldown.data)/2]
			buffUsage.Cooldown.Avg = cooldownSum / float32(len(buffUsage.Cooldown.data))

			////////////////////////////////////////////////////////////////////////////////////////////////////

		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////
	// 마무리 하는 부분

	sort.Slice(
		r.Jobs,
		func(i, k int) bool {
			return ffxiv.JobOrder[r.Jobs[i].Job] > ffxiv.JobOrder[r.Jobs[k].Job]
		},
	)
	for _, job := range r.jobsMap {
		sort.Slice(
			job.Data,
			func(i, k int) bool {
				return job.Data[i].Skill.ID > job.Data[k].Skill.ID
			},
		)
	}

	return r
}
