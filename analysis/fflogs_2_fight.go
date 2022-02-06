package analysis

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sync/atomic"

	"ffxiv_check/cache"
	"ffxiv_check/share"
	"ffxiv_check/share/parallel"
)

func (inst *analysisInstance) updateFights() bool {
	type TodoData struct {
		Hash string

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

		h := fnv.New64a()

		var hash string
		for {
			h.Write(share.S2b(report.ReportID))
			h.Write(sb.Bytes())

			hash = "h" + hex.EncodeToString(h.Sum(nil))
			if _, ok := todoMap[hash]; !ok {
				break
			}
		}

		td := &TodoData{
			Hash:     "h" + hex.EncodeToString(h.Sum(nil)),
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
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"actors"`
		} `json:"masterData"`
	}

	var worked int32
	do := func(hash string, reportData *RespReportData, save bool) {
		td, ok := todoMap[hash]
		if !ok {
			return
		}
		td.retries = 0

		if !td.done {
			td.done = true
			atomic.AddInt32(&worked, 1)
		}

		if reportData == nil {
			return
		}

		for _, fightData := range reportData.Fights {
			if save {
				cache.Report(
					td.ReportID,
					td.FightIDs,
					reportData,
					true,
				)
			}

			sourceId := 0
			for _, friendlyPlayerID := range fightData.FriendlyPlayers {
				for _, actor := range reportData.MasterData.Actors {
					if friendlyPlayerID == actor.ID && actor.Name == inst.CharName {
						sourceId = friendlyPlayerID
					}
				}
				if sourceId != 0 {
					break
				}
			}

			key := fightKey{
				ReportID: td.ReportID,
				FightID:  fightData.ID,
			}
			f, ok := inst.Fights[key]
			if !ok {
				continue
			}

			if sourceId == 0 {
				continue
			}

			f.StartTime = fightData.StartTime
			f.EndTime = fightData.EndTime
			f.SourceID = sourceId
		}
	}

	progress := func() {
		inst.progress(
			"[2 / 3] 전투 정보 가져오는 중... %.2f %%",
			float32(atomic.LoadInt32(&worked))/float32(len(todoList))*100,
		)
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	var respCache RespReportData
	for _, todo := range todoList {
		ok := cache.Report(
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
					pp.Add(work(query))
					query = query[:0]
					qCount++
				}
			}
		}
		if len(query) > 0 {
			pp.Add(work(query))
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
			return false
		}
	}

	progress()

	return true
}
