package analysis

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ffxiv_check/cache"

	fflogs "github.com/RyuaNerin/go-fflogs"
	"github.com/RyuaNerin/go-fflogs/structure"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

const (
	MaxRetries = 3
	Workers    = 8
)

type instance struct {
	inputContext             context.Context
	inputCharName            string
	inputCharServer          string
	inputCharRegion          fflogs.Region
	inputAdditionalPartition []int
	inputJob                 map[string]bool

	encounter []*encounterData

	encounterNamesLock sync.Mutex
	encounterNames     map[int]string

	progress1EncounterWorked int32
	progress1EncounterMax    int
	progress2ReportWorked    int32
	progress2ReportMax       int
	progress3FightWorked     int32
	progress3FightMax        int

	progressString chan string
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
	reportID   string
	fightID    int
	charName   string
	charServer string

	startTime int
	endTime   int

	sourceId int
	job      string

	events []castsEvent
}

type castsEvent struct {
	avilityID   int
	avilityType int
	timestamp   int
}

func (inst *instance) doParallel(
	addToPool func(ctx context.Context, ch chan interface{}),
	work func(ctx context.Context, data interface{}) error,
) error {
	ctx, ctxCancel := context.WithCancel(inst.inputContext)
	defer ctxCancel()

	chanData := make(chan interface{})
	chanError := make(chan error)

	chanAdded := make(chan struct{})
	go func() {
		addToPool(ctx, chanData)
		close(chanAdded)
	}()

	var w sync.WaitGroup
	for i := 0; i < Workers; i++ {
		w.Add(1)
		go func() {
			defer w.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case data, ok := <-chanData:
					if !ok {
						return
					}
					err := work(ctx, data)
					if err != nil {
						select {
						case chanError <- err:
						default:
						}
						return
					}
				}
			}
		}()
	}

	var err error
	select {
	case <-chanAdded:
	case err = <-chanError:
		ctxCancel()
	}
	close(chanData)
	w.Wait()

	return err
}

func (inst *instance) updateReports() error {
	type dataReports struct {
		encData *encounterData
		part    int
	}

	inst.progress1EncounterMax = len(inst.encounter) * (1 + len(inst.inputAdditionalPartition))
	return inst.doParallel(
		func(ctx context.Context, chanJobs chan interface{}) {
			for _, enc := range inst.encounter {
				chanJobs <- &dataReports{
					encData: enc,
					part:    -1,
				}

				for _, partition := range inst.inputAdditionalPartition {
					chanJobs <- &dataReports{
						encData: enc,
						part:    partition,
					}
				}
			}
		},
		func(ctx context.Context, data interface{}) error {
			d := data.(*dataReports)
			return inst.updateReportsWork(ctx, d.encData, d.part)
		},
	)
}

func (inst *instance) updateReportsWork(ctx context.Context, encData *encounterData, part int) error {
	inst.progressString <- fmt.Sprintf(
		"전투 기록 분석 중 %.2f %%",
		float32(atomic.AddInt32(&inst.progress1EncounterWorked, 1))/float32(inst.progress1EncounterMax)*100/3,
	)

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
	var respRaw interface{}
	var respError structure.BaseResponse
	var err error
	for retries := 0; retries < MaxRetries; retries++ {
		err := client.Raw.ParsesCharacter(ctx, &opt, &respRaw)
		if err == nil {
			err = mapstructure.Decode(respRaw, &resp)
			if err == nil {

				break
			}
		}

		err = mapstructure.Decode(respRaw, &respError)
		if err == nil && strings.HasPrefix(respError.Error, "Invalid character") {
			break
		}

		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return errors.WithStack(err)
	}

	encData.reportsLock.Lock()
	defer encData.reportsLock.Unlock()

	for _, ranking := range resp {
		spec := strings.ReplaceAll(ranking.Spec, " ", "")

		_, ok := inst.inputJob[spec]
		if !ok {
			continue
		}

		dic, ok := encData.reports[ranking.ReportID]
		if !ok {
			dic = &reportData{
				reportID:  ranking.ReportID,
				fightData: make(map[int]*fightData),
			}
			encData.reports[ranking.ReportID] = dic
		}

		dic.fightData[ranking.FightID] = &fightData{
			job:        spec,
			reportID:   ranking.ReportID,
			fightID:    ranking.FightID,
			charName:   ranking.CharacterName,
			charServer: ranking.Server,
		}

		inst.encounterNamesLock.Lock()
		inst.encounterNames[ranking.EncounterID] = ranking.EncounterName
		inst.encounterNamesLock.Unlock()
	}

	return nil
}

func (inst *instance) updateFights() error {
	inst.progress2ReportMax = 0
	for _, enc := range inst.encounter {
		inst.progress2ReportMax += len(enc.reports)
	}

	return inst.doParallel(
		func(ctx context.Context, chanJobs chan interface{}) {
			for _, enc := range inst.encounter {
				for _, report := range enc.reports {
					chanJobs <- report
				}
			}
		},
		func(ctx context.Context, data interface{}) error {
			return inst.updateFightsWork(ctx, data.(*reportData))
		},
	)
}

func (inst *instance) updateFightsWork(ctx context.Context, report *reportData) error {
	inst.progressString <- fmt.Sprintf(
		"전투 기록 분석 중 %.2f %%",
		100/3+float32(atomic.AddInt32(&inst.progress2ReportWorked, 1))/float32(inst.progress2ReportMax)*100/3,
	)

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
			return errors.WithStack(err)
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

				if reportFriendly.Server == nil || *reportFriendly.Server != fight.charServer {
					continue
				}
				if reportFriendly.Name != fight.charName {
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
			return errors.WithStack(err)
		}

		cache.Report(report.reportID, &resp, true)
	}

	return nil
}

func (inst *instance) updateEvents() error {
	inst.progress3FightMax = 0
	for _, enc := range inst.encounter {
		for _, report := range enc.reports {
			inst.progress3FightMax += len(report.fightData)
		}
	}

	return inst.doParallel(
		func(ctx context.Context, chanJobs chan interface{}) {
			for _, enc := range inst.encounter {
				for _, report := range enc.reports {
					for _, fight := range report.fightData {
						if fight.sourceId == 0 {
							continue
						}

						chanJobs <- fight
					}
				}
			}
		},
		func(ctx context.Context, data interface{}) error {
			return inst.updateEventsWork(ctx, data.(*fightData))
		},
	)
}

func (inst *instance) updateEventsWork(ctx context.Context, fight *fightData) error {
	inst.progressString <- fmt.Sprintf(
		"전투 기록 분석 중 %.2f %%",
		2*100/3+float32(atomic.AddInt32(&inst.progress3FightWorked, 1))/float32(inst.progress3FightMax)*100/3,
	)

	startTime := fight.startTime

	opt := fflogs.ReportEventsOptions{
		Code:     fight.reportID,
		Sourceid: &fight.sourceId,
		Start:    &startTime,
		End:      &fight.endTime,
	}

	var resp Events

	for {
		if !cache.CastsEvent(fight.reportID, fight.fightID, fight.sourceId, startTime, fight.endTime, &resp, false) {
			var err error
			for retries := 0; retries < MaxRetries; retries++ {
				err = client.Raw.ReportEventsCasts(ctx, &opt, &resp)
				if err == nil {
					break
				}
				time.Sleep(5 * time.Second)
			}
			if err != nil {
				return errors.WithStack(err)
			}

			cache.CastsEvent(fight.reportID, fight.fightID, fight.sourceId, startTime, fight.endTime, &resp, true)
		}

		len := len(fight.events)
		new := make([]castsEvent, len, len+resp.Count)
		copy(fight.events, new)
		fight.events = new

		for _, event := range resp.Events {
			if event.Type == "cast" {
				fight.events = append(
					fight.events,
					castsEvent{
						avilityID:   event.Ability.GUID,
						avilityType: event.Ability.Type,
						timestamp:   event.Timestamp,
					},
				)
			}
		}

		if resp.NextPageTimestamp == nil {
			break
		}
		startTime = *resp.NextPageTimestamp
	}

	return nil
}
