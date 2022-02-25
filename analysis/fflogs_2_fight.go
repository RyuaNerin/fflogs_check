package analysis

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"ffxiv_check/share/parallel"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

func (inst *analysisInstance) updateFights() bool {
	log.Printf("updateFights %s@%s\n", inst.InpCharName, inst.InpCharServer)
	type TodoData struct {
		Hash     string
		ReportID string
		FightIDs string

		retries int
		done    bool
	}

	todoList := make([]*TodoData, 0, len(inst.Reports))
	todoMap := make(map[string]*TodoData, len(inst.Reports))
	var sb bytes.Buffer
	for _, report := range inst.Reports {
		f := true
		sb.Reset()
		for _, fight := range report.Fights {
			if f {
				fmt.Fprintf(&sb, "%d", fight.FightID)
				f = false
			} else {
				fmt.Fprintf(&sb, ",%d", fight.FightID)
			}
		}

		td := &TodoData{
			Hash:     fmt.Sprintf("h%d", len(todoList)),
			ReportID: report.ReportID,
			FightIDs: sb.String(),
		}
		todoList = append(todoList, td)
		todoMap[td.Hash] = td
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	type RespReportData struct {
		Fights []struct {
			ID              int   `json:"id"`
			StartTime       int   `json:"startTime"`
			EndTime         int   `json:"endTime"`
			FriendlyPlayers []int `json:"friendlyPlayers"`
		} `json:"fights"`
		MasterData struct {
			Actors []struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				SubType string `json:"subType"`
			} `json:"actors"`
		} `json:"masterData"`
	}

	var worked int32
	do := func(hash string, resp *RespReportData, save bool) {
		td, ok := todoMap[hash]
		if !ok {
			return
		}
		td.retries = 0

		if !td.done {
			td.done = true
			atomic.AddInt32(&worked, 1)
		}

		if resp == nil {
			return
		}
		if save {
			cacheReport(
				td.ReportID,
				td.FightIDs,
				resp,
				true,
			)
		}

		for _, respFight := range resp.Fights {
			sourceId := 0
			for _, respFightPlayerID := range respFight.FriendlyPlayers {
				for _, respActor := range resp.MasterData.Actors {
					if respFightPlayerID == respActor.ID && respActor.Name == inst.InpCharName {
						sourceId = respFightPlayerID
					}
				}
				if sourceId != 0 {
					break
				}
			}

			key := fightKey{
				ReportID: td.ReportID,
				FightID:  respFight.ID,
			}
			fight, ok := inst.Fights[key]
			if !ok {
				continue
			}

			// 아이디나 서버를 변경 한 경우
			if sourceId == 0 {
				// 직겹이 없는 경우 직업으로 검색이 가능함.
				for _, respFightPlayerID := range respFight.FriendlyPlayers {
					for _, respActor := range resp.MasterData.Actors {
						if respFightPlayerID == respActor.ID && respActor.SubType == fight.Job {
							if sourceId != 0 {
								// 직겹으로 갔음!!!
								sourceId = 0
								break
							} else {
								sourceId = respFightPlayerID
							}
						}
					}
				}

				// 그래도 못 찾은 경우 어쩔 수 없음.
				if sourceId == 0 {
					continue
				}
			}

			fight.DoneSummary = true
			fight.StartTime = respFight.StartTime
			fight.EndTime = respFight.EndTime
			fight.SourceID = sourceId
		}
	}

	progressPercent := func() float32 {
		return float32(atomic.LoadInt32(&worked)) / float32(len(todoList)) * 100
	}
	progress := func() {
		p := progressPercent()
		log.Printf("updateFights %s@%s (%.2f %%)\n", inst.InpCharName, inst.InpCharServer, p)
		inst.progress("[2 / 3] 전투 정보 가져오는 중... %.2f %%", p)
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	var respCache RespReportData
	for _, todo := range todoList {
		ok := cacheReport(
			todo.ReportID,
			todo.FightIDs,
			&respCache,
			false,
		)
		if ok {
			do(todo.Hash, &respCache, false)
		}
	}
	progress()

	////////////////////////////////////////////////////////////////////////////////////////////////////

	pp := parallel.New(workers)

	work := func(queryOrig []*TodoData) func(ctx context.Context) error {
		query := make([]*TodoData, len(queryOrig))
		copy(query, queryOrig)

		return func(ctx context.Context) error {
			if ctx.Err() != nil {
				return nil
			}

			var resp struct {
				Data struct {
					ReportData map[string]*RespReportData `json:"reportData"`
				} `json:"data"`
			}

			err := inst.try(func() error { return inst.callGraphQl(ctx, tmplReportSummary, query, &resp) })
			if err != nil {
				sentry.CaptureException(err)
				fmt.Printf("%+v\n", errors.WithStack(err))
				return err
			}

			for hash, reportData := range resp.Data.ReportData {
				do(hash, reportData, true)
			}
			progress()

			return nil
		}
	}

	query := make([]*TodoData, 0, maxSummary)
	for {
		pp.Reset(inst.ctx)

		qCount := 0
		for _, todo := range todoList {
			if todo.retries < 3 && !todo.done {
				todo.retries++
				query = append(query, todo)

				if len(query) == maxSummary {
					pp.Do(work(query))
					query = query[:0]
					qCount++
				}
			}
		}
		if len(query) > 0 {
			pp.Do(work(query))
			query = query[:0]
			qCount++
		}

		if qCount == 0 {
			break
		}

		err := pp.Wait()
		if err != nil {
			return false
		}
	}

	// 미완료되면 실패
	for _, todo := range todoList {
		if !todo.done {
			sentry.CaptureException(errors.Errorf(
				"Report: %s (%s) / %s@%s / retries : %d", todo.ReportID, todo.FightIDs, inst.InpCharName, inst.InpCharServer, todo.retries,
			))
			return false
		}
	}

	progress()

	return true
}
