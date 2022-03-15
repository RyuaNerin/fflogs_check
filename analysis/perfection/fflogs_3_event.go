package perfection

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync/atomic"

	"ffxiv_check/analysis"
	"ffxiv_check/share/parallel"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

func (inst *analysisInstance) updateEvents() bool {
	log.Printf("updateEvents %s@%s\n", inst.InpCharName, inst.InpCharServer)
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

		StartTime int
		EndTime   int

		Casts       TodoFightEvent
		Buffs       TodoFightEvent
		Deaths      TodoFightEvent
		AttacksDone bool
		DebuffsDone bool

		done    bool
		retries int
	}

	todoList := make([]*TodoFight, 0, len(inst.Reports))
	todoMap := make(map[string]*TodoFight, len(inst.Reports))
	for _, report := range inst.Reports {
		for _, fight := range report.Fights {
			if !fight.DoneSummary {
				continue
			}

			td := &TodoFight{
				Hash: fmt.Sprintf("h%d", len(todoList)),

				ReportID: report.ReportID,
				FightID:  fight.FightID,
				SourceID: fight.SourceID,

				StartTime: fight.StartTime,
				EndTime:   fight.EndTime,

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
				AttacksDone: fight.Job != "Paladin", // 충의 계산용...
				DebuffsDone: false,
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
			TargetID      int    `json:"targetID"`
		}
		NextPageTimestamp *int `json:"nextPageTimestamp"`
	}

	type RespReportData struct {
		Casts   *respReportEventData `json:"casts"`
		Buffs   *respReportEventData `json:"buffs"`
		Deaths  *respReportEventData `json:"deaths"`
		Attacks *struct {
			Data struct {
				Entries []struct {
					Uses int `json:"uses"` // Attacks
				} `json:"entries"`
			} `json:"data"`
		} `json:"attacks"`
		Debuffs *struct {
			Data struct {
				Auras []struct {
					GUID        int `json:"guid"`        // Debuffs
					TotalUptime int `json:"totalUptime"` // Debuffs
					TotalUses   int `json:"totalUses"`   // Debuffs
				} `json:"auras"`
			} `json:"data"`
		} `json:"debuffs"`
	}

	var worked int32
	do := func(hash string, resp *RespReportData, save bool) {
		td, ok := todoMap[hash]
		if !ok {
			return
		}

		if resp == nil {
			td.Casts.Done = true
			td.Buffs.Done = true
			td.Deaths.Done = true
			if !td.done {
				td.done = true
				atomic.AddInt32(&worked, 1)
			}
			return
		}

		if save {
			cacheCastsEvent(
				td.ReportID,
				td.FightID,
				td.SourceID,
				td.Casts.StartTime, td.Casts.EndTime,
				td.Buffs.StartTime, td.Buffs.EndTime,
				td.Deaths.StartTime, td.Deaths.EndTime,
				resp,
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

		if resp.Casts != nil {
			if resp.Casts.NextPageTimestamp == nil {
				td.Casts.Done = true
			} else {
				td.Casts.StartTime = *resp.Casts.NextPageTimestamp
				td.retries = 0
			}

			l := len(fightInfo.Casts)
			if cap(fightInfo.Casts) < l+len(resp.Casts.Data) {
				newArr := make([]analysisEvent, l, l+len(resp.Casts.Data))
				copy(newArr, fightInfo.Casts)
				fightInfo.Casts = newArr
			}

			for _, event := range resp.Casts.Data {
				switch event.Type {
				case "cast":
					fightInfo.Casts = append(
						fightInfo.Casts,
						analysisEvent{
							gameID:    event.AbilityGameID,
							timestamp: event.Timestamp - td.StartTime,
						},
					)
				}
			}
		}

		if resp.Buffs != nil {
			if resp.Buffs.NextPageTimestamp == nil {
				td.Buffs.Done = true
			} else {
				td.Buffs.StartTime = *resp.Buffs.NextPageTimestamp
				td.retries = 0
			}

			l := len(fightInfo.Buffs)
			if cap(fightInfo.Buffs) < l+len(resp.Buffs.Data) {
				newArr := make([]analysisBuff, l, l+len(resp.Buffs.Data))
				copy(newArr, fightInfo.Buffs)
				fightInfo.Buffs = newArr
			}

			for _, event := range resp.Buffs.Data {
				if event.TargetID != fightInfo.SourceID {
					continue
				}

				switch event.Type {
				case "applybuff":
					fightInfo.Buffs = append(
						fightInfo.Buffs,
						analysisBuff{
							timestamp: event.Timestamp - td.StartTime,
							gameID:    event.AbilityGameID,
							removed:   false,
						},
					)
				case "removebuff":
					fightInfo.Buffs = append(
						fightInfo.Buffs,
						analysisBuff{
							timestamp: event.Timestamp - td.StartTime,
							gameID:    event.AbilityGameID,
							removed:   true,
						},
					)
				}
			}
		}

		if resp.Deaths != nil {
			if resp.Deaths.NextPageTimestamp == nil {
				td.Deaths.Done = true
			} else {
				td.Deaths.StartTime = *resp.Deaths.NextPageTimestamp
				td.retries = 0
			}

			l := len(fightInfo.Deaths)
			if cap(fightInfo.Deaths) < l+len(resp.Deaths.Data) {
				newArr := make([]analysisDeath, l, l+len(resp.Deaths.Data))
				copy(newArr, fightInfo.Deaths)
				fightInfo.Deaths = newArr
			}

			for _, event := range resp.Deaths.Data {
				switch event.Type {
				case "death":
					fightInfo.Deaths = append(
						fightInfo.Deaths,
						analysisDeath{
							timestamp: event.Timestamp - td.StartTime,
						},
					)
				}
			}
		}

		if resp.Attacks != nil {
			td.AttacksDone = true

			for _, entries := range resp.Attacks.Data.Entries {
				fightInfo.AutoAttacks += entries.Uses
			}
		}

		if resp.Debuffs != nil {
			td.DebuffsDone = true

			for _, auras := range resp.Debuffs.Data.Auras {
				switch auras.GUID {
				case 1002092: // 주는 피해량 감소 (칠흑 재생편)
					fallthrough
				case 1002911: // 주는 피해량 감소 (효월 변옥편)
					fightInfo.Debuff.ReduceDamange.count = auras.TotalUses
					fightInfo.Debuff.ReduceDamange.uptime = auras.TotalUptime
				}
			}
		}

		if !td.done {
			if td.Casts.Done && td.Buffs.Done && td.Deaths.Done && td.AttacksDone && td.DebuffsDone {
				td.done = true
				fightInfo.DoneEvents = true
				atomic.AddInt32(&worked, 1)
			}
		}
	}

	progressPercent := func() float32 {
		return float32(atomic.LoadInt32(&worked)) / float32(len(todoList)) * 100
	}
	progress := func() {
		p := progressPercent()
		log.Printf("updateEvents %s@%s (%.2f %%)\n", inst.InpCharName, inst.InpCharServer, progressPercent())
		inst.progress("[3 / 3] 전투 정보 분석 중... %.2f %%", p)
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	var respCache RespReportData
	for _, todo := range todoList {
		for {
			ok := cacheCastsEvent(
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

			err := analysis.CallGraphQL(ctx, tmplReportCastsEvents, query, &resp)
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

	query := make([]*TodoFight, 0, maxEvents)
	for {
		pp.Reset(inst.ctx)

		qCount := 0
		for _, todo := range todoList {
			if todo.retries < 3 && !todo.done {
				todo.retries++
				query = append(query, todo)

				if len(query) == maxEvents {
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
				"Report: %s (fight: %d) / %s@%s / retries : %d", todo.ReportID, todo.FightID, inst.InpCharName, inst.InpCharServer, todo.retries,
			))
			return false
		}
	}

	for _, fight := range inst.Fights {
		if !fight.DoneEvents || !fight.DoneSummary {
			continue
		}

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
