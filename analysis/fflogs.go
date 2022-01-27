package analysis

import (
	"context"
	"sync"
	"time"

	"ffxiv_check/cache"
	"ffxiv_check/share/semaphore"

	fflogs "github.com/RyuaNerin/go-fflogs"
	"github.com/pkg/errors"
)

const MaxRetries = 3

var sema = semaphore.New(16)

type instance struct {
	inputContext             context.Context
	inputCharName            string
	inputCharServer          string
	inputCharRegion          fflogs.Region
	inputAdditionalPartition []int

	encounter []*encounterData

	opt *AnalyzeOptions
}

type encounterData struct {
	zoneID      int
	encounterID int

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
	name      string
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
		len(inst.encounter)*(1+len(inst.inputAdditionalPartition)),
		func(w *sync.WaitGroup, ctx context.Context, ch chan error) {
			for _, enc := range inst.encounter {
				w.Add(1)
				go inst.updateReportsWork(w, ctx, ch, enc, -1)

				for _, partition := range inst.inputAdditionalPartition {
					w.Add(1)
					go inst.updateReportsWork(w, ctx, ch, enc, partition)
				}
			}
		},
	)
}

func (inst *instance) updateReportsWork(w *sync.WaitGroup, ctx context.Context, ch chan error, encData *encounterData, part int) {
	defer w.Done()

	sema.Acquire()
	defer sema.Release()

	opt := fflogs.ParsesCharacterOptions{
		CharacterName: inst.inputCharName,
		ServerName:    inst.inputCharServer,
		ServerRegion:  fflogs.RegionKR,
		Zone:          &encData.zoneID,
		Encounter:     &encData.encounterID,
	}
	if part != -1 {
		opt.Partition = &part
	}

	var resp []CharacterRanking
	var err error
	for retries := 0; retries < MaxRetries; retries++ {
		err := client.Raw.ParsesCharacter(ctx, &opt, &resp)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		ch <- errors.WithStack(err)
		return
	}

	encData.reportsLock.Lock()
	defer encData.reportsLock.Unlock()

	for _, ranking := range resp {
		dic, ok := encData.reports[ranking.ReportID]
		if !ok {
			dic = &reportData{
				reportID:  ranking.ReportID,
				fightData: make(map[int]*fightData),
			}
			encData.reports[ranking.ReportID] = dic
		}

		dic.fightData[ranking.FightID] = &fightData{
			job:      ranking.Spec,
			reportID: ranking.ReportID,
			fightID:  ranking.FightID,
		}
	}
}

func (inst *instance) updateFights() error {
	count := 0
	for _, enc := range inst.encounter {
		count += len(enc.reports)
	}

	return inst.doParallel(
		count,
		func(w *sync.WaitGroup, ctx context.Context, ch chan error) {
			for _, enc := range inst.encounter {
				for _, report := range enc.reports {
					w.Add(1)
					go inst.updateFightsWork(w, ctx, ch, report)
				}
			}
		},
	)
}

func (inst *instance) updateFightsWork(w *sync.WaitGroup, ctx context.Context, ch chan error, report *reportData) {
	defer w.Done()

	sema.Acquire()
	defer sema.Release()

	opt := fflogs.ReportFightsOptions{
		Code: report.reportID,
	}

	var resp Report
	if !cache.Report(report.reportID, &resp, false) {
		var err error
		for retries := 0; retries < MaxRetries; retries++ {
			err := client.Raw.ReportFights(ctx, &opt, &resp)
			if err == nil {
				break
			}
			time.Sleep(5 * time.Second)
		}
		if err != nil {
			ch <- errors.WithStack(err)
			return
		}

		cache.Report(report.reportID, &resp, true)
	}

	for {
		needToFound := len(report.fightData)

		for fightId, fight := range report.fightData {
			for _, reportFight := range resp.Fights {
				if reportFight.ID == fightId {
					fight.startTime = reportFight.StartTime
					fight.endTime = reportFight.EndTime

					needToFound--
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

		if needToFound == 0 {
			break
		}

		var err error
		for retries := 0; retries < MaxRetries; retries++ {
			err := client.Raw.ReportFights(ctx, &opt, &resp)
			if err == nil {
				break
			}
			time.Sleep(5 * time.Second)
		}
		if err != nil {
			ch <- errors.WithStack(err)
			return
		}

		cache.Report(report.reportID, &resp, true)
	}
}

func (inst *instance) updateEvents() error {
	totalFights := 0
	for _, enc := range inst.encounter {
		for _, report := range enc.reports {
			totalFights += len(report.fightData)
		}
	}

	return inst.doParallel(
		totalFights,
		func(w *sync.WaitGroup, ctx context.Context, ch chan error) {
			for _, enc := range inst.encounter {
				for _, report := range enc.reports {
					for _, fight := range report.fightData {
						w.Add(1)
						go inst.updateEventsWork(w, ctx, ch, fight)
					}
				}
			}
		},
	)
}

func (inst *instance) updateEventsWork(w *sync.WaitGroup, ctx context.Context, ch chan error, fight *fightData) {
	defer w.Done()

	sema.Acquire()
	defer sema.Release()

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
			var err error
			for retries := 0; retries < MaxRetries; retries++ {
				err = client.Raw.ReportEventsCasts(ctx, &opt, &resp)
				if err == nil {
					break
				}
				time.Sleep(5 * time.Second)
			}
			if err != nil {
				ch <- errors.WithStack(err)
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
				name:      event.Ability.Name,
				timestamp: event.Timestamp,
			}
		}

		if resp.NextPageTimestamp == nil {
			break
		}
		startTime = *resp.NextPageTimestamp
	}
}
