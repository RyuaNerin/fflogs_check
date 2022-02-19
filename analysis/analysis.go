package analysis

import (
	"context"
	"os"

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
	CharName             string   `json:"char_name"`
	CharServer           string   `json:"char_server"`
	CharRegion           string   `json:"char_region"`
	Encouters            []int    `json:"encounters"`
	AdditionalPartitions []int    `json:"partitions"`
	Jobs                 []string `json:"jobs"`
}

func Analyze(ctx context.Context, progress func(p string), opt *AnalyzeOptions) (stat *Statistic, ok bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, ctxCancel := context.WithCancel(ctx)

	inst := analysisInstance{
		ctx: ctx,

		InpCharName:     opt.CharName,
		InpCharServer:   opt.CharServer,
		InpCharRegion:   opt.CharRegion,
		InpCharJobs:     make(map[string]bool, len(opt.Jobs)),
		InpEncounterIDs: make([]int, len(opt.Encouters)),

		Reports: make(map[string]*analysisReport),
		Fights:  make(map[fightKey]*analysisFight),

		encounterNames: make(map[int]string, len(opt.Encouters)),
		encounterRanks: make(map[int]*analysisRank, 1+len(opt.Jobs)),

		progressString: make(chan string),
	}
	defer close(inst.progressString)

	if opt.CharRegion == "kr" {
		inst.skillSets = &ffxiv.Korea
	} else {
		inst.skillSets = &ffxiv.Global
	}

	copy(inst.InpEncounterIDs, opt.Encouters)

	for _, job := range opt.Jobs {
		_, ok := ffxiv.JobOrder[job]
		if ok {
			inst.InpCharJobs[job] = true
		}
	}

	inst.InpAdditionalPartition = append(inst.InpAdditionalPartition, opt.AdditionalPartitions...)

	chanDone := make(chan struct{}, 1)
	go func() {
		defer close(chanDone)

		for {
			select {
			case <-ctx.Done():
				return

			case s := <-inst.progressString:
				progress(s)
			}
		}
	}()

	succ := inst.updateReports()
	if succ && inst.charState == StatisticStateNormal {
		succ = inst.updateFights()
		if succ {
			succ = inst.updateEvents()
		}
	}

	if succ {
		stat = inst.buildReport()
		ok = true
	}

	ctxCancel()
	<-chanDone

	return
}
