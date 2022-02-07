package analysis

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

func (inst *analysisInstance) updateReports() bool {
	inst.progress("[1 / 3] 전투 기록 가져오는 중...")

	var resp struct {
		Data struct {
			WorldData map[string]struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"worldData"`
			CharacterData struct {
				Character map[string]struct {
					Ranks []struct {
						Spec   string `json:"spec"`
						Report struct {
							Code    string `json:"code"`
							FightID int    `json:"fightID"`
						} `json:"report"`
					} `json:"ranks"`
				} `json:"character"`
			} `json:"characterData"`
		} `json:"data"`
	}

	err := inst.try(func() error { return inst.callGraphQl(inst.ctx, tmplEncounterRankings, inst, &resp) })
	if err != nil {
		fmt.Printf("%+v\n", errors.WithStack(err))
		return false
	}

	for _, encData := range resp.Data.WorldData {
		inst.encounterNames[encData.ID] = encData.Name
	}

	for key, value := range resp.Data.CharacterData.Character {
		encStr := reEnc.FindStringSubmatch(key)
		if len(encStr) != 2 {
			continue
		}

		encId, err := strconv.Atoi(encStr[1])
		if err != nil {
			continue
		}

		for _, rank := range value.Ranks {
			report, ok := inst.Reports[rank.Report.Code]
			if !ok {
				report = &analysisReport{
					ReportID: rank.Report.Code,
				}
				inst.Reports[rank.Report.Code] = report
			}

			key := fightKey{
				ReportID: rank.Report.Code,
				FightID:  rank.Report.FightID,
			}
			value := &analysisFight{
				ReportID:    rank.Report.Code,
				FightID:     rank.Report.FightID,
				Job:         rank.Spec,
				EncounterID: encId,
			}

			inst.Fights[key] = value
			report.Fights = append(report.Fights, value)
		}
	}

	return len(inst.Fights) > 0
}
