package analysis

import (
	"context"
	"os"
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
	CharName             string   `json:"char_name"`
	CharServer           string   `json:"char_server"`
	CharRegion           string   `json:"char_region"`
	Encouters            []int    `json:"encounters"`
	AdditionalPartitions []int    `json:"partitions"`
	Jobs                 []string `json:"jobs"`
}

func Analyze(ctx context.Context, progress func(p string), opt *AnalyzeOptions) (stat *Statistics, ok bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, ctxCancel := context.WithCancel(ctx)

	inst := analysisInstance{
		ctx: ctx,

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

	if opt.CharRegion == "kr" {
		inst.skillSets = ffxiv.Korea
	} else {
		inst.skillSets = ffxiv.Global
	}

	copy(inst.EncounterIDs, opt.Encouters)

	for _, job := range opt.Jobs {
		_, ok := ffxiv.JobOrder[job]
		if ok {
			inst.CharJobs[job] = true
		}
	}

	inst.AdditionalPartition = append(inst.AdditionalPartition, opt.AdditionalPartitions...)

	chanDone := make(chan struct{}, 1)
	go func() {
		defer close(chanDone)

		nextMessage := time.Now()
		for {
			select {
			case <-ctx.Done():
				return

			case s := <-inst.progressString:
				if time.Now().Before(nextMessage) {
					continue
				}

				progress(s)
				nextMessage = time.Now().Add(250 * time.Millisecond)
			}
		}
	}()

	if inst.updateReports() {
		if inst.updateFights() {
			if inst.updateEvents() {
				stat = inst.buildReport()
				ok = true
			}
		}
	}

	ctxCancel()
	<-chanDone

	return
}
