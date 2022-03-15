package allstar

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"ffxiv_check/analysis"
	"ffxiv_check/ffxiv"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

func Do(ctx context.Context, reqData *analysis.RequestData, progress func(p string), buf *bytes.Buffer) bool {
	tmplData := tmplData{
		CharName:   reqData.CharName,
		CharServer: reqData.CharServer,
		UpdatedAt:  time.Now().Format("2006-01-02 15:04:05"),
		State:      statisticStateInvalid,
		jobsMap:    make(map[string]*tmplDataJob),
	}

	if !doStat(ctx, reqData, progress, &tmplData) {
		return false
	}

	err := tmplResult.Execute(buf, tmplData)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return false
	}

	return true
}

func doStat(ctx context.Context, reqData *analysis.RequestData, progress func(p string), allstarData *tmplData) bool {
	if reqData.CharRegion != "kr" {
		allstarData.State = statisticStateInvalid
		return true
	}

	preset, ok := presetMap[reqData.Preset]
	if !ok {
		allstarData.State = statisticStateInvalid
		return true
	}

	allstarData.ZoneName = preset.Name
	allstarData.ShowAllstar = preset.UseAllstarRank

	ok = reqData.CheckOptionValidation()
	if !ok {
		allstarData.State = statisticStateInvalid
		return true
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, ctxCancel := context.WithCancel(ctx)

	inst := analysisInstance{
		ctx:      ctx,
		tmplData: allstarData,
		Preset:   preset,

		CharName:   reqData.CharName,
		CharServer: reqData.CharServer,
		Jobs:       make(map[string]bool, len(reqData.Jobs)),

		progressString: make(chan string),
	}
	defer close(inst.progressString)

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

	for _, job := range reqData.Jobs {
		_, ok := ffxiv.JobOrder[job]
		if ok {
			inst.Jobs[strings.ToUpper(job)] = true
		}
	}

	res := false
	if inst.UpdateKrEncounterRdps() {
		if inst.tmplData.State != statisticStateNormal {
			res = true
		} else {
			if inst.UpdateKrEncounterRank() {
				if inst.UpdateGlobalRank() {
					inst.UpdateMapToSlice()
					res = true
				}
			}
		}
	}

	ctxCancel()
	<-chanDone

	return res
}
