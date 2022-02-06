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
	"ffxiv_check/share/parallel"

	"github.com/pkg/errors"
)

func (inst *analysisInstance) updateEvents() bool {
	type TodoFightEvent struct {
		Done      bool
		StartTime int
		EndTime   int
	}

	type TodoFight struct {
		Hash string

		ReportID string
		FightID  int
		SourceID int

		Casts  TodoFightEvent
		Buffs  TodoFightEvent
		Deaths TodoFightEvent

		done    bool
		retries int

		fightStartTime int
	}

	todoList := make([]*TodoFight, 0, len(inst.Reports))
	todoMap := make(map[string]*TodoFight, len(inst.Reports))
	for _, report := range inst.Reports {
		for _, fight := range report.Fights {
			h := fnv.New64a()

			var hash string
			for {
				h.Write(share.S2b(report.ReportID))
				fmt.Fprint(h, fight.FightID)

				hash = "h" + hex.EncodeToString(h.Sum(nil))
				if _, ok := todoMap[hash]; !ok {
					break
				}
			}

			td := &TodoFight{
				Hash:     hash,
				ReportID: report.ReportID,
				FightID:  fight.FightID,
				SourceID: fight.SourceID,
				Casts: TodoFightEvent{
					StartTime: fight.StartTime,
					EndTime:   fight.EndTime,
				},
				Buffs: TodoFightEvent{
					StartTime: fight.StartTime,
					EndTime:   fight.EndTime,
				},
				Deaths: TodoFightEvent{
					StartTime: fight.StartTime,
					EndTime:   fight.EndTime,
				},
				fightStartTime: fight.StartTime,
			}
			todoList = append(todoList, td)
			todoMap[td.Hash] = td
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	type respReportEventData struct {
		Data []struct {
			Timestamp     int    `json:"timestamp"`
			Type          string `json:"type"`
			AbilityGameID int    `json:"abilityGameID"`
		}
		NextPageTimestamp *int `json:"nextPageTimestamp"`
	}

	type RespReportData struct {
		Casts  *respReportEventData `json:"casts"`
		Buffs  *respReportEventData `json:"buffs"`
		Deaths *respReportEventData `json:"deaths"`
	}

	var worked int32
	do := func(hash string, reportData *RespReportData, save bool) {
		td, ok := todoMap[hash]
		if !ok {
			return
		}

		td.retries = 0

		if reportData == nil {
			td.done = true
			td.Casts.Done = true
			td.Buffs.Done = true
			td.Deaths.Done = true
			atomic.AddInt32(&worked, 1)
			return
		}

		if save {
			cache.CastsEvent(
				td.ReportID,
				td.FightID,
				td.SourceID,
				td.Casts.StartTime, td.Casts.EndTime,
				td.Buffs.StartTime, td.Buffs.EndTime,
				td.Deaths.StartTime, td.Deaths.EndTime,
				reportData,
				true,
			)
		}

		fightKey := fightKey{
			ReportID: td.ReportID,
			FightID:  td.FightID,
		}
		fightInfo, ok := inst.Fights[fightKey]
		if !ok {
			return
		}

		if reportData.Casts != nil {
			if reportData.Casts.NextPageTimestamp == nil {
				td.Casts.Done = true
			} else {
				td.Casts.StartTime = *reportData.Casts.NextPageTimestamp
			}

			l := len(fightInfo.Casts)
			if cap(fightInfo.Casts) < l+len(reportData.Casts.Data) {
				newArr := make([]analysisEvent, l, l+len(reportData.Casts.Data))
				copy(newArr, fightInfo.Casts)
				fightInfo.Casts = newArr
			}

			for _, event := range reportData.Casts.Data {
				switch event.Type {
				case "cast":
					fightInfo.Casts = append(
						fightInfo.Casts,
						analysisEvent{
							gameID:    event.AbilityGameID,
							timestamp: event.Timestamp - td.fightStartTime,
						},
					)
				}
			}
		}

		if reportData.Buffs != nil {
			if reportData.Buffs.NextPageTimestamp == nil {
				td.Buffs.Done = true
			} else {
				td.Buffs.StartTime = *reportData.Buffs.NextPageTimestamp
			}

			l := len(fightInfo.Buffs)
			if cap(fightInfo.Buffs) < l+len(reportData.Buffs.Data) {
				newArr := make([]analysisBuff, l, l+len(reportData.Buffs.Data))
				copy(newArr, fightInfo.Buffs)
				fightInfo.Buffs = newArr
			}

			for _, event := range reportData.Buffs.Data {
				switch event.Type {
				case "applybuff":
					fightInfo.Buffs = append(
						fightInfo.Buffs,
						analysisBuff{
							timestamp: event.Timestamp - td.fightStartTime,
							gameID:    event.AbilityGameID,
							removed:   false,
						},
					)
				case "removebuff":
					fightInfo.Buffs = append(
						fightInfo.Buffs,
						analysisBuff{
							timestamp: event.Timestamp - td.fightStartTime,
							gameID:    event.AbilityGameID,
							removed:   true,
						},
					)
				}
			}
		}

		if reportData.Deaths != nil {
			if reportData.Deaths.NextPageTimestamp == nil {
				td.Deaths.Done = true
			} else {
				td.Deaths.StartTime = *reportData.Deaths.NextPageTimestamp
			}

			l := len(fightInfo.Deaths)
			if cap(fightInfo.Deaths) < l+len(reportData.Deaths.Data) {
				newArr := make([]analysisDeath, l, l+len(reportData.Deaths.Data))
				copy(newArr, fightInfo.Deaths)
				fightInfo.Deaths = newArr
			}

			for _, event := range reportData.Deaths.Data {
				switch event.Type {
				case "death":
					fightInfo.Deaths = append(
						fightInfo.Deaths,
						analysisDeath{
							timestamp: event.Timestamp - td.fightStartTime,
						},
					)
				}
			}
		}

		if !td.done {
			if td.Casts.Done && td.Buffs.Done && td.Deaths.Done {
				td.done = true
				atomic.AddInt32(&worked, 1)
			}
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
				todo.Casts.StartTime, todo.Casts.EndTime,
				todo.Buffs.StartTime, todo.Buffs.EndTime,
				todo.Deaths.StartTime, todo.Deaths.EndTime,
				&respCache,
				false,
			)
			if !ok {
				break
			}
			do(todo.Hash, &respCache, false)

			if todo.done {
				break
			}
		}
	}
	progress()

	////////////////////////////////////////////////////////////////////////////////////////////////////

	pp := parallel.New(workers)

	work := func(queryOrig []*TodoFight) func(ctx context.Context) error {
		query := make([]*TodoFight, len(queryOrig))
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

			err := inst.try(func() error { return inst.callGraphQl(ctx, tmplReportCastsEvents, query, &resp) })
			if err != nil {
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

	query := make([]*TodoFight, 0, maxEvents)
	for {
		pp.Reset(inst.ctx)

		qCount := 0
		for _, todo := range todoList {
			if todo.retries < 3 && !todo.done {
				todo.retries++
				query = append(query, todo)

				if len(query) == maxEvents {
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

	for _, fight := range inst.Fights {
		sort.Slice(
			fight.Casts,
			func(i, k int) bool {
				return fight.Casts[i].timestamp < fight.Casts[k].timestamp
			},
		)
		sort.Slice(
			fight.Buffs,
			func(i, k int) bool {
				return fight.Buffs[i].timestamp < fight.Buffs[k].timestamp
			},
		)
		sort.Slice(
			fight.Deaths,
			func(i, k int) bool {
				return fight.Deaths[i].timestamp < fight.Deaths[k].timestamp
			},
		)
	}

	progress()

	return true
}
