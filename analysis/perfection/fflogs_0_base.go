package perfection

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"ffxiv_check/analysis"
	"ffxiv_check/ffxiv"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

func Do(ctx context.Context, reqData *analysis.RequestData, progress func(p string), buf *bytes.Buffer) bool {
	stat := statistic{
		CharName:   reqData.CharName,
		CharServer: reqData.CharServer,
		CharRegion: reqData.CharRegion,
		UpdatedAt:  time.Now().Format("2006-01-02 15:04:05"),
		State:      statisticStateInvalid,
	}

	if !doStat(ctx, reqData, progress, &stat) {
		return false
	}

	err := tmplResult.Execute(buf, stat)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return false
	}

	return true
}

func doStat(ctx context.Context, reqData *analysis.RequestData, progress func(p string), stat *statistic) bool {
	preset, ok := presets[reqData.Preset]
	if !ok {
		stat.State = statisticStateInvalid
		return true
	}

	ok = reqData.CheckOptionValidation()
	if !ok {
		stat.State = statisticStateInvalid
		return true
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, ctxCancel := context.WithCancel(ctx)

	inst := analysisInstance{
		ctx: ctx,

		InpCharName:     reqData.CharName,
		InpCharServer:   reqData.CharServer,
		InpCharRegion:   reqData.CharRegion,
		InpEncounterIDs: make([]int, len(preset.Enc)),
		InpDifficulty:   preset.Diff,

		Reports: make(map[string]*analysisReport),
		Fights:  make(map[fightKey]*analysisFight),

		encounterNames: make(map[int]string, len(preset.Enc)),
		encounterRanks: make(map[int]*analysisRank, 1+len(ffxiv.JobOrder)),

		progressString: make(chan string),

		stat: stat,
	}
	defer close(inst.progressString)

	inst.gameData = ffxiv.GameDataMap[preset.Version]

	copy(inst.InpEncounterIDs, preset.Enc)

	switch reqData.CharRegion {
	case "kr":
		inst.InpAdditionalPartition = append(inst.InpAdditionalPartition, preset.Part.Korea...)
	case "gl":
		inst.InpAdditionalPartition = append(inst.InpAdditionalPartition, preset.Part.Global...)
	}

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

	res := false
	succ := inst.updateReports()
	if succ {
		if inst.charState == statisticStateNormal {
			succ = inst.updateFights()
			if succ {
				succ = inst.updateEvents()
			}
		}
		if succ {
			inst.buildReport()
			res = true
		}
	}

	ctxCancel()
	<-chanDone

	return res
}
