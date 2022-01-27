package analysis

import (
	"context"
	"ffxiv_check/cache"
	"sync"

	fflogs "github.com/RyuaNerin/go-fflogs"
)

type instance struct {
	inputContext             context.Context
	inputCharName            string
	inputCharServer          string
	inputCharRegion          fflogs.Region
	inputZone                int
	inputEncounterId         int
	inputAdditionalPartition []int

	reportsLock sync.Mutex
	reports     map[string]*reportData
}

type reportData struct {
	reportID string

	fightData map[int]*fightData
}

type fightData struct {
	reportID string
	fightID  int

	startTime int
	endTime   int

	sourceId int
	job      string

	events []castsEvent
}

type castsEvent struct {
	id        int
	timestamp int
}

func (inst *instance) doParallel(
	count int,
	f func(w *sync.WaitGroup, ctx context.Context, ch chan error),
) error {
	context, contextCancel := context.WithCancel(inst.inputContext)

	chanError := make(chan error, count)

	var w sync.WaitGroup
	f(&w, context, chanError)

	go func() {
		w.Wait()
		chanError <- nil
	}()

	err := <-chanError
	contextCancel()
	w.Wait()
	return err
}

func (inst *instance) updateReports() error {
	return inst.doParallel(
		1+len(inst.inputAdditionalPartition),
		func(w *sync.WaitGroup, ctx context.Context, ch chan error) {
			w.Add(1)
			go inst.updateReportsWork(w, ctx, ch, nil)

			for _, partition := range inst.inputAdditionalPartition {
				v := new(int)
				*v = partition

				w.Add(1)
				go inst.updateReportsWork(w, ctx, ch, v)
			}
		},
	)
}

func (inst *instance) updateReportsWork(w *sync.WaitGroup, ctx context.Context, ch chan error, part *int) {
	defer w.Done()

	opt := fflogs.ParsesCharacterOptions{
		CharacterName: inst.inputCharName,
		ServerName:    inst.inputCharServer,
		ServerRegion:  fflogs.RegionKR,
		Zone:          &inst.inputZone,
		Encounter:     &inst.inputEncounterId,
		Partition:     part,
	}

	var resp []CharacterRanking
	err := client.Raw.ParsesCharacter(ctx, &opt, &resp)
	if err != nil {
		ch <- err
		return
	}

	inst.reportsLock.Lock()
	defer inst.reportsLock.Unlock()

	for _, ranking := range resp {
		dic, ok := inst.reports[ranking.ReportID]
		if !ok {
			dic = &reportData{
				reportID:  ranking.ReportID,
				fightData: make(map[int]*fightData),
			}
			inst.reports[ranking.ReportID] = dic
		}

		dic.fightData[ranking.FightID] = &fightData{
			job:      ranking.Spec,
			reportID: ranking.ReportID,
			fightID:  ranking.FightID,
		}
	}
}

func (inst *instance) updateFights() error {
	return inst.doParallel(
		len(inst.reports),
		func(w *sync.WaitGroup, ctx context.Context, ch chan error) {
			for _, report := range inst.reports {
				w.Add(1)
				go inst.updateFightsWork(w, ctx, ch, report)
			}
		},
	)
}

func (inst *instance) updateFightsWork(w *sync.WaitGroup, ctx context.Context, ch chan error, report *reportData) {
	defer w.Done()

	opt := fflogs.ReportFightsOptions{
		Code: report.reportID,
	}

	var resp Report
	if !cache.Report(report.reportID, &resp, false) {
		err := client.Raw.ReportFights(ctx, &opt, &resp)
		if err != nil {
			ch <- err
			return
		}

		cache.Report(report.reportID, &resp, true)
	}

	for fightId, fight := range report.fightData {
		for _, reportFight := range resp.Fights {
			if reportFight.ID == fightId {
				fight.startTime = reportFight.StartTime
				fight.endTime = reportFight.EndTime
				break
			}
		}

		for _, reportFriendly := range resp.Friendlies {
			notFound := true
			for _, reportFriendlyFights := range reportFriendly.Fights {
				if fightId == reportFriendlyFights.ID {
					notFound = false
					break
				}
			}
			if notFound {
				continue
			}

			if reportFriendly.Server == nil || *reportFriendly.Server != inst.inputCharServer {
				continue
			}
			if reportFriendly.Name != inst.inputCharName {
				continue
			}

			fight.sourceId = reportFriendly.ID
			fight.job = reportFriendly.Job

			break
		}
	}
}

func (inst *instance) updateEvents() error {
	totalFights := 0
	for _, report := range inst.reports {
		totalFights += len(report.fightData)
	}

	return inst.doParallel(
		totalFights,
		func(w *sync.WaitGroup, ctx context.Context, ch chan error) {
			for _, report := range inst.reports {
				for _, fight := range report.fightData {
					w.Add(1)
					go inst.updateEventsWork(w, ctx, ch, fight)
				}
			}
		},
	)
}

func (inst *instance) updateEventsWork(w *sync.WaitGroup, ctx context.Context, ch chan error, fight *fightData) {
	defer w.Done()

	startTime := fight.startTime

	opt := fflogs.ReportEventsOptions{
		Code:     fight.reportID,
		Sourceid: &fight.sourceId,
		Start:    &startTime,
		End:      &fight.endTime,
	}

	var resp Events

	for {
		if !cache.CastsEvent(fight.reportID, fight.fightID, fight.sourceId, startTime, &resp, false) {
			err := client.Raw.ReportEventsCasts(ctx, &opt, &resp)
			if err != nil {
				ch <- err
				return
			}

			cache.CastsEvent(fight.reportID, fight.fightID, fight.sourceId, startTime, &resp, true)
		}

		len := len(fight.events)
		new := make([]castsEvent, len+resp.Count)
		copy(fight.events, new)
		fight.events = new

		for i, event := range resp.Events {
			fight.events[len+i] = castsEvent{
				id:        event.Ability.GUID,
				timestamp: event.Timestamp,
			}
		}

		if resp.NextPageTimestamp == nil {
			break
		}
		startTime = *resp.NextPageTimestamp
	}
}
