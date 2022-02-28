package analysis

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

var (
	reCharacter = regexp.MustCompile(`^([^_]+)_(\d+)(_\d+)?$`)
)

func (inst *analysisInstance) updateReports() bool {
	log.Printf("updateReports %s@%s\n", inst.InpCharName, inst.InpCharServer)
	inst.progress("[1 / 3] 전투 기록 가져오는 중...")

	var resp struct {
		Data struct {
			WorldData map[string]struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"worldData"`
			CharacterData struct {
				CharData *struct {
					ID     int  `json:"id"`
					Hidden bool `json:"hidden"`
				} `json:"char"`
				CharEncounter map[string]struct {
					Ranks []struct {
						RankPercent float64 `json:"rankPercent"`
						Amount      float64 `json:"amount"`
						Spec        string  `json:"spec"`
						Report      struct {
							Code    string `json:"code"`
							FightID int    `json:"fightID"`
						} `json:"report"`
					} `json:"ranks"`
				} `json:"char_encounter"`
			} `json:"characterData"`
		} `json:"data"`
	}

	err := inst.try(func() error { return inst.callGraphQl(inst.ctx, tmplEncounterRankings, inst, &resp) })
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return false
	}

	if resp.Data.CharacterData.CharData == nil {
		inst.charState = StatisticStateNotFound
		return true
	}

	if resp.Data.CharacterData.CharData.Hidden {
		inst.charState = StatisticStateHidden
		return true
	}

	inst.charID = resp.Data.CharacterData.CharData.ID
	inst.charState = StatisticStateNormal

	for _, encData := range resp.Data.WorldData {
		inst.encounterNames[encData.ID] = encData.Name
	}

	addRank := func(job string, arrSize int, encId int, isDps bool, rank float32, amount float32) {
		rankJob, ok := inst.encounterRanks[encId]
		if !ok {
			rankJob = &analysisRank{
				Dps: make(map[string]*analysisRankData, 1+len(inst.InpCharJobs)),
				Hps: make(map[string]*analysisRankData, 1+len(inst.InpCharJobs)),
			}
			inst.encounterRanks[encId] = rankJob
		}

		d := rankJob.Dps
		if !isDps {
			d = rankJob.Hps
		}

		rankData, ok := d[job]
		if !ok {
			rankData = &analysisRankData{
				Data: make([]fflogsRankData, 0, arrSize),
			}
			d[job] = rankData
		}

		v := fflogsRankData{
			Rank:   float32(rank),
			Amount: float32(amount),
		}

		rankData.Data = append(rankData.Data, v)
	}

	for key, charData := range resp.Data.CharacterData.CharEncounter {
		encStr := reCharacter.FindStringSubmatch(key)
		if len(encStr) != 4 {
			continue
		}

		isDps := encStr[1] == "dps"

		encId, err := strconv.Atoi(encStr[2])
		if err != nil {
			continue
		}

		for _, rank := range charData.Ranks {
			_, ok := inst.InpCharJobs[strings.ToUpper(rank.Spec)]
			if !ok {
				continue
			}

			addRank(rank.Spec, len(charData.Ranks), encId, isDps, float32(rank.RankPercent), float32(rank.Amount))

			////////////////////////////////////////////////////////////////////////////////

			report, ok := inst.Reports[rank.Report.Code]
			if !ok {
				report = &analysisReport{
					ReportID: rank.Report.Code,
				}
				inst.Reports[rank.Report.Code] = report
			}

			if isDps {
				key := fightKey{
					ReportID: rank.Report.Code,
					FightID:  rank.Report.FightID,
				}
				value := &analysisFight{
					ReportID:    rank.Report.Code,
					FightID:     rank.Report.FightID,
					Job:         rank.Spec,
					EncounterID: encId,
					skillData:   make(map[int]*analysisFightSkill),
				}

				inst.Fights[key] = value
				report.Fights = append(report.Fights, value)
			}
		}
	}

	for _, report := range inst.Reports {
		sort.Slice(
			report.Fights,
			func(i, k int) bool {
				return report.Fights[i].FightID < report.Fights[k].FightID
			},
		)
	}

	if len(inst.Fights) == 0 {
		inst.charState = StatisticStateNoLog
		return true
	}

	return true
}
