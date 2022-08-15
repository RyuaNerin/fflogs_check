package hps

import (
	"context"
	"fmt"
	"text/template"

	"ffxiv_check/analysis"
	"ffxiv_check/share"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

var (
	tmplCharacterData = template.Must(template.ParseFiles("analysis/hps/query/CharacterData.tmpl"))
)

// map[ReportCode][]FightID
func getCharacterData(ctx context.Context, charName, charServer, charRegion string, preset preset) (charID int, hidden bool, reportData map[string][]int, encounterNameMap map[int]string, ok bool) {
	reqData := struct {
		CharName                string
		CharServer              string
		CharRegion              string
		EncounterIDList         []int
		AdditionalPartitionList []int
		Difficulty              int
	}{
		CharName:        charName,
		CharServer:      charServer,
		CharRegion:      charRegion,
		EncounterIDList: preset.Enc,
		Difficulty:      preset.Diff,
	}
	if charRegion == "kr" {
		reqData.AdditionalPartitionList = preset.Part.Korea
	} else {
		reqData.AdditionalPartitionList = preset.Part.Global
	}

	var respData struct {
		Data struct {
			WorldData map[string]struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"worldData"`
			CharacterData struct {
				CharInfo *struct {
					ID     int  `json:"id"`
					Hidden bool `json:"hidden"`
				} `json:"char_info"`
				CharRankings map[string]struct {
					Ranks []struct {
						Spec   string `json:"spec"`
						Report struct {
							Spec    string `json:"spec"`
							Code    string `json:"code"`
							FightID int    `json:"fightID"`
						} `json:"report"`
					} `json:"ranks"`
				} `json:"char_rankings"`
			} `json:"characterData"`
		} `json:"data"`
	}

	err := analysis.CallGraphQL(ctx, tmplCharacterData, &reqData, &respData)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		ok = false
		return
	}

	// 캐릭터를 찾을 수 없는 경우 null...
	if respData.Data.CharacterData.CharInfo == nil {
		charID = 0
		ok = true
		return
	}

	reportData = make(map[string][]int)
	for _, rankings := range respData.Data.CharacterData.CharRankings {
		for _, rank := range rankings.Ranks {
			if !share.StringInSortedSlice(specHealerList, rank.Spec) {
				continue
			}
			reportData[rank.Report.Code] = append(reportData[rank.Report.Code], rank.Report.FightID)
		}
	}

	charID = respData.Data.CharacterData.CharInfo.ID
	hidden = respData.Data.CharacterData.CharInfo.Hidden

	encounterNameMap = make(map[int]string, len(reqData.EncounterIDList))
	for _, elem := range respData.Data.WorldData {
		encounterNameMap[elem.ID] = elem.Name
	}

	ok = true
	return
}
