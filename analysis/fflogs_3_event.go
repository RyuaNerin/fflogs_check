package analysis

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sort"
	"sync/atomic"

	"ffxiv_check/cache"
	"ffxiv_check/share"

	"github.com/gammazero/workerpool"
)

func (inst *analysisInstance) updateEvents() bool {
	type TodoFight struct {
		Hash string

		ReportID string
		FightID  int
		SourceID int

		StartTime int
		EndTime   int

		fightStartTime int
		retries        int
		done           bool
	}

	todoList := make([]*TodoFight, 0, len(inst.Reports))
	todoMap := make(map[string]*TodoFight, len(inst.Reports))
	for _, report := range inst.Reports {
		for _, fight := range report.Fights {
			h := fnv.New64()
			h.Write(share.S2b(report.ReportID))
			fmt.Fprint(h, fight.FightID)

			td := &TodoFight{
				Hash:           "h" + hex.EncodeToString(h.Sum(nil)),
				ReportID:       report.ReportID,
				FightID:        fight.FightID,
				SourceID:       fight.SourceID,
				StartTime:      fight.StartTime,
				EndTime:        fight.EndTime,
				fightStartTime: fight.StartTime,
			}
			todoList = append(todoList, td)
			todoMap[td.Hash] = td
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	type RespReportData struct {
		Data []struct {
			Timestamp     int    `json:"timestamp"`
			Type          string `json:"type"`
			AbilityGameID int    `json:"abilityGameID"`
		}
		NextPageTimestamp *int `json:"nextPageTimestamp"`
	}

	var worked int32
	do := func(hash string, reportData *RespReportData, save bool) {
		td, ok := todoMap[hash]
		if !ok {
			return
		}

		if reportData == nil {
			td.done = true
			atomic.AddInt32(&worked, 1)
			return
		}

		if save {
			cache.CastsEvent(
				td.ReportID,
				td.FightID,
				td.SourceID,
				td.StartTime,
				td.EndTime,
				reportData,
				true,
			)
		}

		if reportData.NextPageTimestamp == nil {
			if !td.done {
				td.done = true
				atomic.AddInt32(&worked, 1)
			}
		} else {
			td.StartTime = *reportData.NextPageTimestamp
		}

		fightKey := fightKey{
			ReportID: td.ReportID,
			FightID:  td.FightID,
		}
		fightInfo, ok := inst.Fights[fightKey]
		if !ok {
			return
		}

		l := len(fightInfo.Events)
		if cap(fightInfo.Events) < l+len(reportData.Data) {
			newArr := make([]analysisEvent, l, l+len(reportData.Data))
			copy(newArr, fightInfo.Events)
			fightInfo.Events = newArr
		}

		for _, event := range reportData.Data {
			if event.Type == "cast" {
				continue
			}

			fightInfo.Events = append(
				fightInfo.Events,
				analysisEvent{
					avilityID: event.AbilityGameID,
					timestamp: event.Timestamp - td.fightStartTime,
				},
			)
		}
	}

	progress := func() {
		inst.progress(
			"[3 / 3] 전투 정보 분석 중... %.2f %%",
			float32(atomic.LoadInt32(&worked))/float32(len(todoList))*100,
		)
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	var respCache RespReportData
	for _, todo := range todoList {
		for {
			ok := cache.CastsEvent(
				todo.ReportID,
				todo.FightID,
				todo.SourceID,
				todo.StartTime,
				todo.EndTime,
				&respCache,
				false,
			)
			if !ok {
				break
			}
			do(todo.Hash, &respCache, false)

			if respCache.NextPageTimestamp == nil {
				break
			}
			todo.StartTime = *respCache.NextPageTimestamp
		}
	}
	progress()

	////////////////////////////////////////////////////////////////////////////////////////////////////

	wpCtx, wpCtxCancel := context.WithCancel(inst.ctx)
	wp := workerpool.New(workers)

	work := func(queryOrig []*TodoFight) func() {
		query := make([]*TodoFight, len(queryOrig))
		copy(query, queryOrig)

		return func() {
			if wpCtx.Err() != nil {
				return
			}

			var resp struct {
				Data struct {
					ReportData map[string]*RespReportData `json:"reportData"`
				} `json:"data"`
			}

			err := inst.try(func() error { return inst.callGraphQl(wpCtx, tmplReportCastsEvents, query, &resp) })
			if err != nil {
				wpCtxCancel()
				wp.Stop()
				return
			}

			for hash, reportData := range resp.Data.ReportData {
				do(hash, reportData, true)
			}
			progress()
		}
	}

	query := make([]*TodoFight, 0, maxSummary)
	for {
		qCount := 0
		for _, todo := range todoList {
			if !todo.done && todo.retries < 3 {
				todo.retries++
				query = append(query, todo)

				if len(query) == maxSummary {
					wp.Submit(work(query))
					query = query[:0]
					qCount++
				}
			}
		}
		if len(query) > 0 {
			wp.Submit(work(query))
			query = query[:0]
			qCount++
		}

		if qCount == 0 {
			break
		}

		wp.StopWait()
		err := wpCtx.Err()
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

	for _, fight := range inst.Fights {
		sort.Slice(
			fight.Events,
			func(i, k int) bool {
				return fight.Events[i].timestamp < fight.Events[k].timestamp
			},
		)
	}

	progress()

	return true
}