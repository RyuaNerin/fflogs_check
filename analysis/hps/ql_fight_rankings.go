package hps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"text/template"

	"ffxiv_check/analysis"
	"ffxiv_check/share/parallel"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

const (
	maxFightRankings = 20
)

var (
	tmplFightRankings = template.Must(template.ParseFiles("analysis/hps/query/FightRankings.tmpl"))
)

type rankingData struct {
	CharID      int     `json:"char-id"`
	CharName    string  `json:"name"`
	SpecIdx     int     `json:"spec-idx"`
	RankPercent float32 `json:"rankPercent"`
}
type rankingDataPair struct {
	EncounterID int    `json:"encounter-id"`
	ReportCode  string `json:"report-code"`
	FightID     int    `json:"fight-id"`

	Healer0 rankingData `json:"h1"`
	Healer1 rankingData `json:"h2"`
}

func getFightRankings(ctx context.Context, reportData map[string][]int, progressFunc func(desc string)) (rankingDataPairList []*rankingDataPair, ok bool) {
	type TodoData struct {
		Key        string
		ReportCode string
		FightIDs   string

		retries int
		done    bool
	}

	type RankingData struct {
		FightID   int `json:"fightID"`
		Encounter struct {
			ID int `json:"id"`
		} `json:"encounter"`
		Roles struct {
			Healer struct {
				Characters []struct {
					ID     int    `json:"id"`
					Name   string `json:"name"`
					Server struct {
						Name string `json:"name"`
					} `json:"server"`
					Spec        string          `json:"class"` // spec이 아니라 class 로 제공함. 주의
					RankPercent json.RawMessage `json:"rankPercent"`
				} `json:"characters"`
			} `json:"healers"`
		} `json:"roles"`
	}

	var rankingDataPairListLock sync.Mutex

	var progressRemain atomic.Int32
	progressRemain.Swap(int32(len(reportData)))

	progress := func() {
		p := 100 - float64(progressRemain.Load())/float64(len(reportData))*100
		progressFunc(fmt.Sprintf("[2 / 3] 전투 정보 가져오는 중... %.2f %%", p))
	}
	progress()

	////////////////////////////////////////////////////////////

	var todoLock sync.Mutex
	todoList := make([]*TodoData, 0, len(reportData))
	todoMap := make(map[string]*TodoData)
	var sb bytes.Buffer
	for reportCode, fightIDs := range reportData {
		sb.Reset()
		for _, fightID := range fightIDs {
			// 캐시에서 가져온 적 있나 확인
			var cache rankingDataPair
			if cacheFightRankings(reportCode, fightID, &cache, false) {
				rankingDataPairList = append(rankingDataPairList, &cache)
				continue
			}

			if sb.Len() != 0 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, "%d", fightID)
		}

		if sb.Len() == 0 {
			progressRemain.Add(-1)
			continue
		}

		td := &TodoData{
			Key:        fmt.Sprintf("h%d", len(todoList)),
			ReportCode: reportCode,
			FightIDs:   sb.String(),
		}
		todoList = append(todoList, td)
		todoMap[td.Key] = td
	}
	progress()

	////////////////////////////////////////////////////////////

	if len(todoList) > 0 {
		// 병렬 작업
		pp := parallel.New(workers)

		ppFunc := func(query []*TodoData) func(ctx context.Context) error {
			queryCopy := make([]*TodoData, len(query))
			copy(queryCopy, query)

			return func(ctx context.Context) error {
				if ctx.Err() != nil {
					return nil
				}

				var respData struct {
					Data struct {
						ReportData map[string]struct {
							Rankings struct {
								Data []RankingData `json:"data"`
							} `json:"rankings"`
						} `json:"reportData"`
					} `json:"data"`
				}

				err := analysis.CallGraphQL(ctx, tmplFightRankings, queryCopy, &respData)
				if err != nil {
					sentry.CaptureException(err)
					fmt.Printf("%+v\n", errors.WithStack(err))
					return err
				}

				for key, reportData := range respData.Data.ReportData {
					todoLock.Lock()
					td, ok := todoMap[key]
					if !ok {
						todoLock.Unlock()
						continue
					}

					td.done = true
					delete(todoMap, key)
					progressRemain.Add(-1)
					todoLock.Unlock()

					for _, ranking := range reportData.Rankings.Data {
						if len(ranking.Roles.Healer.Characters) != 2 {
							continue
						}

						rp0, err := strconv.ParseFloat(string(ranking.Roles.Healer.Characters[0].RankPercent), 32)
						if err != nil {
							continue
						}
						rp1, err := strconv.ParseFloat(string(ranking.Roles.Healer.Characters[1].RankPercent), 32)
						if err != nil {
							continue
						}

						spec0 := sort.SearchStrings(specHealerList, ranking.Roles.Healer.Characters[0].Spec)
						spec1 := sort.SearchStrings(specHealerList, ranking.Roles.Healer.Characters[1].Spec)
						if spec0 >= len(specHealerList) || spec1 >= len(specHealerList) {
							continue
						}

						value := &rankingDataPair{
							EncounterID: ranking.Encounter.ID,
							ReportCode:  td.ReportCode,
							FightID:     ranking.FightID,
							Healer0: rankingData{
								SpecIdx:     spec0,
								RankPercent: float32(rp0),
								CharID:      ranking.Roles.Healer.Characters[0].ID,
								CharName: fmt.Sprintf(
									"%s@%s",
									ranking.Roles.Healer.Characters[0].Name,
									ranking.Roles.Healer.Characters[0].Server.Name,
								),
							},
							Healer1: rankingData{
								SpecIdx:     spec1,
								RankPercent: float32(rp1),
								CharID:      ranking.Roles.Healer.Characters[1].ID,
								CharName: fmt.Sprintf(
									"%s@%s",
									ranking.Roles.Healer.Characters[1].Name,
									ranking.Roles.Healer.Characters[1].Server.Name,
								),
							},
						}

						cacheFightRankings(td.ReportCode, ranking.FightID, value, true)

						rankingDataPairListLock.Lock()
						rankingDataPairList = append(rankingDataPairList, value)
						rankingDataPairListLock.Unlock()
					}
				}
				progress()

				return nil
			}
		}

		ppQuery := make([]*TodoData, 0, maxFightRankings)
		for {
			pp.Reset(ctx)

			doCount := 0
			for _, todo := range todoList {
				if todo.retries > 3 || todo.done {
					continue
				}

				todo.retries++
				ppQuery = append(ppQuery, todo)
				if len(ppQuery) > maxFightRankings {
					pp.Do(ppFunc(ppQuery))
					ppQuery = ppQuery[:0]
					doCount++
				}
			}
			if len(ppQuery) > 0 {
				pp.Do(ppFunc(ppQuery))
				ppQuery = ppQuery[:0]
				doCount++
			}

			if doCount == 0 {
				break
			}

			err := pp.Wait()
			if err != nil {
				ok = false
				return
			}
		}
	}

	ok = true
	return
}
