package hps

import (
	"bytes"
	"context"
	"ffxiv_check/analysis"
	"ffxiv_check/share"
	"fmt"
	"sort"
	"text/template"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

var (
	tmplResult = template.Must(
		template.
			New("template.tmpl.htm").
			Funcs(share.TemplateFuncMap).
			ParseFiles("analysis/hps/template.tmpl.htm"),
	)
)

type resultState string

const (
	resultStateNormal   resultState = "normal"
	resultStateHidden   resultState = "hidden"
	resultStateNotFound resultState = "notfound"
	resultStateInvalid  resultState = "invalid"
	resultStateNoLog    resultState = "nolog"
)

type ResultData struct {
	State     resultState `json:"status"`
	UpdatedAt string      `json:"updated_at"`

	CharName   string `json:"char-name"`
	CharServer string `json:"char-server"`
	CharRegion string `json:"char-region"`

	FFLogsLink string `json:"fflogs-link"`

	JobList []string `json:"spec-list"`

	Encounter    []*ResultEncounterData       `json:"encounter"`
	encounterMap map[int]*ResultEncounterData `json:"-"`
}
type ResultEncounterData struct {
	EncounterID   int    `json:"encounter-id"`
	EncounterName string `json:"encounter-name"`

	Kills int

	Data [][][]ResultValueData `json:"data"`
}
type ResultValueData struct {
	ScoreMe float32 `json:"me"`
	ScorePn float32 `json:"pn"`
	NameMe  string  `json:"mename"`
	NamePn  string  `json:"pnname"`

	ReportCode string `json:"reportcode"`
	FightID    int    `json:"fightid"`
}

func Do(ctx context.Context, reqData *analysis.RequestData, progress func(p string), buf *bytes.Buffer) bool {
	resultData := ResultData{
		State:        resultStateInvalid,
		CharName:     reqData.CharName,
		CharServer:   reqData.CharServer,
		CharRegion:   reqData.CharRegion,
		UpdatedAt:    time.Now().Format("2006-01-02 15:04:05"),
		JobList:      specHealerList,
		encounterMap: make(map[int]*ResultEncounterData, 5),
	}

	if !doInner(ctx, reqData, progress, &resultData) {
		return false
	}

	err := tmplResult.Execute(buf, &resultData)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return false
	}

	return true
}

func doInner(ctx context.Context, reqData *analysis.RequestData, progressFunc func(p string), resultData *ResultData) bool {
	if !reqData.CheckOptionValidation() {
		resultData.State = resultStateInvalid
		return true
	}

	preset, ok := presets[reqData.Preset]
	if !ok {
		resultData.State = resultStateInvalid
		return true
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	////////////////////////////////////////////////////////////////////////////////////////////////////

	progressFunc("[1 / 3] 전투 기록 가져오는 중...")
	charID, charHidden, reportData, encounterNameMap, ok := getCharacterData(
		ctx,
		reqData.CharName,
		reqData.CharServer,
		reqData.CharRegion,
		preset,
	)
	if !ok {
		return false
	}
	if charID == 0 {
		resultData.State = resultStateNotFound
		return true
	}
	if charHidden {
		resultData.State = resultStateHidden
		return true
	}
	if len(reportData) == 0 {
		resultData.State = resultStateNoLog
		return true
	}

	switch reqData.CharRegion {
	case "kr":
		resultData.FFLogsLink = fmt.Sprintf("https://ko.fflogs.com/character/id/%d", charID)
	default:
		resultData.FFLogsLink = fmt.Sprintf("https://www.fflogs.com/character/id/%d", charID)
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	chanProgressDone := make(chan struct{}, 1)
	chanProgressString := make(chan string, 1)
	go func() {
		defer close(chanProgressDone)

		for {
			select {
			case <-ctx.Done():
				return

			case s := <-chanProgressString:
				progressFunc(s)
			}
		}
	}()
	progress := func(s string) {
		if ctx.Err() == nil {
			select {
			case <-ctx.Done():
			case chanProgressString <- s:
			default:
			}
		}
	}

	fightRankingList, ok := getFightRankings(ctx, reportData, progress)
	if !ok {
		return false
	}
	ctxCancel()
	<-chanProgressDone

	////////////////////////////////////////////////////////////////////////////////////////////////////

	encounterNameMap[0] = "모든 보스"
	newEncounter := func(encounterID int) *ResultEncounterData {
		encounter := &ResultEncounterData{
			EncounterID:   encounterID,
			EncounterName: encounterNameMap[encounterID],
			Data:          make([][][]ResultValueData, len(specHealerList)),
		}
		for idx0 := range specHealerList {
			encounter.Data[idx0] = make([][]ResultValueData, len(specHealerList))
		}
		return encounter
	}

	for idx, fightRanking := range fightRankingList {
		progress(fmt.Sprintf(
			"[3 / 3] 전투 정보 분석 중... %.2f %%",
			float64(idx)/float64(len(fightRankingList))*100,
		))

		var playerMe, playerOther rankingData

		if fightRanking.Healer0.CharID == charID && fightRanking.Healer1.CharID != charID {
			playerMe = fightRanking.Healer0
			playerOther = fightRanking.Healer1
		} else if fightRanking.Healer0.CharID != charID && fightRanking.Healer1.CharID == charID {
			playerMe = fightRanking.Healer1
			playerOther = fightRanking.Healer0
		} else {
			continue
		}

		encounterAll, ok := resultData.encounterMap[0]
		if !ok {
			encounterAll = newEncounter(0)
			resultData.encounterMap[0] = encounterAll
		}
		encounterAll.Kills++

		encounterSpc, ok := resultData.encounterMap[fightRanking.EncounterID]
		if !ok {
			encounterSpc = newEncounter(fightRanking.EncounterID)
			resultData.encounterMap[fightRanking.EncounterID] = encounterSpc
		}
		encounterSpc.Kills++

		data := ResultValueData{
			ScoreMe:    playerMe.RankPercent,
			ScorePn:    playerOther.RankPercent,
			NameMe:     playerMe.CharName,
			NamePn:     playerOther.CharName,
			ReportCode: fightRanking.ReportCode,
			FightID:    fightRanking.FightID,
		}

		encounterAll.Data[playerMe.SpecIdx][playerOther.SpecIdx] = append(encounterAll.Data[playerMe.SpecIdx][playerOther.SpecIdx], data)
		encounterSpc.Data[playerMe.SpecIdx][playerOther.SpecIdx] = append(encounterSpc.Data[playerMe.SpecIdx][playerOther.SpecIdx], data)
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	// Map To List
	resultData.Encounter = make([]*ResultEncounterData, 0, len(resultData.encounterMap))
	for _, encounter := range resultData.encounterMap {
		resultData.Encounter = append(resultData.Encounter, encounter)
	}
	sort.Slice(
		resultData.Encounter,
		func(i, k int) bool {
			return resultData.Encounter[i].EncounterID < resultData.Encounter[k].EncounterID
		},
	)

	resultData.State = resultStateNormal
	return true
}
