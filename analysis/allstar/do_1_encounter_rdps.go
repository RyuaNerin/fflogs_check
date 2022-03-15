package allstar

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"ffxiv_check/analysis"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

var (
	reCharacter = regexp.MustCompile(`^dps_(\d+)_(\d+)$`)
)

func (inst *analysisInstance) UpdateKrEncounterRdps() bool {
	log.Printf("UpdateKrEncounterRdps %s@%s\n", inst.CharName, inst.CharServer)
	inst.progress("[1 / 3] 전투 기록 가져오는 중...")

	var resp struct {
		Data struct {
			CharacterData struct {
				CharData *struct {
					ID     int  `json:"id"`
					Hidden bool `json:"hidden"`
				} `json:"char"`
				CharEncounter map[string]struct {
					Ranks []struct {
						TodayPercent float32 `json:"todayPercent"`
						Amount       float32 `json:"amount"`
						Spec         string  `json:"spec"`
					} `json:"ranks"`
				} `json:"char_encounter"`
			} `json:"characterData"`
		} `json:"data"`
	}

	err := analysis.CallGraphQL(inst.ctx, tmplEncounterRankings, inst, &resp)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return false
	}

	if resp.Data.CharacterData.CharData == nil {
		inst.tmplData.State = statisticStateNotFound
		return true
	}

	if resp.Data.CharacterData.CharData.Hidden {
		inst.tmplData.State = statisticStateHidden
		return true
	}

	inst.tmplData.State = statisticStateNormal

	inst.tmplData.FFLogsLink = fmt.Sprintf("https://ko.fflogs.com/character/id/%d", resp.Data.CharacterData.CharData.ID)

	logCount := 0
	for key, charData := range resp.Data.CharacterData.CharEncounter {
		encStr := reCharacter.FindStringSubmatch(key)
		if len(encStr) != 3 {
			continue
		}

		encId, err := strconv.Atoi(encStr[1])
		if err != nil {
			continue
		}

		partID, err := strconv.Atoi(encStr[2])
		if err != nil {
			continue
		}

		for _, rank := range charData.Ranks {
			_, ok := inst.Jobs[strings.ToUpper(rank.Spec)]
			if !ok {
				continue
			}

			logCount++

			jobData, ok := inst.tmplData.jobsMap[rank.Spec]
			if !ok {
				jobData = &tmplDataJob{
					Job:           rank.Spec,
					partitionsMap: make(map[int]*tmplDataPartition, len(inst.Preset.Partition)),
				}
				inst.tmplData.jobsMap[rank.Spec] = jobData
			}

			partData, ok := jobData.partitionsMap[partID]
			if !ok {
				part := inst.Preset.PartitionMap[partID]
				partData = &tmplDataPartition{
					PartitionIDKorea:  part.Korea,
					PartitionIDGlobal: part.Global,
					PartitionName:     part.Name,

					encountersMap: make(map[int]*tmplDataEncounter, len(inst.Preset.Encounter)),
				}
				jobData.partitionsMap[partID] = partData
			}

			encData, ok := partData.encountersMap[encId]
			if !ok {
				encData = &tmplDataEncounter{
					EncounterID: encId,
				}
				partData.encountersMap[encId] = encData
			}

			if encData.Rdps < rank.Amount {
				encData.Rdps = rank.Amount
			}
		}
	}

	if logCount == 0 {
		inst.tmplData.State = statisticStateNoLog
		return true
	}

	return true
}
