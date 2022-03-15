package allstar

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"ffxiv_check/analysis"
	"ffxiv_check/share/parallel"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

func (inst *analysisInstance) UpdateKrEncounterRank() bool {
	log.Printf("UpdateKrEncounterRank %s@%s\n", inst.CharName, inst.CharServer)

	type TodoData struct {
		Key       string
		Partition int
		ZoneID    int
		Spec      string

		retries int
		done    bool
	}

	var todoList []*TodoData
	todoMap := make(map[string]*TodoData)

	for _, partData := range inst.tmplData.partitionsMap {
		for _, jobData := range partData.jobsMap {
			td := &TodoData{
				Key:       fmt.Sprintf("h%d", len(todoList)),
				Partition: partData.PartitionIDKorea,
				ZoneID:    inst.Preset.Zone,
				Spec:      jobData.Job,
			}
			todoList = append(todoList, td)
			todoMap[td.Key] = td
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	type RespZoneRanking struct {
		AllStars []struct {
			Points      float32 `json:"points"`
			Rank        int     `json:"rank"`
			RankPercent float32 `json:"rankPercent"`
		} `json:"allStars"`
		Rankings []struct {
			Encounter struct {
				ID int `json:"id"`
			} `json:"encounter"`
			RankPercent *float32 `json:"rankPercent"`
			TotalKills  int      `json:"totalKills"`
			Allstar     *struct {
				Points      float32  `json:"points"`
				Rank        IntV     `json:"rank"`
				RankPercent Float32V `json:"rankPercent"`
			} `json:"allStars"`
		} `json:"rankings"`
	}

	var worked int32
	do := func(hash string, resp *RespZoneRanking, save bool) {
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
			if inst.Preset.PartitionMap[td.Partition].Locked {
				cacheEncounterRank(
					inst.CharName,
					inst.CharServer,
					td.ZoneID,
					td.Partition,
					td.Spec,
					&resp,
					true,
				)
			}
		}

		partData := inst.tmplData.partitionsMap[td.Partition]

		jobData, ok := partData.jobsMap[td.Spec]
		if !ok {
			jobData = &tmplDataJob{
				Job:           td.Spec,
				encountersMap: make(map[int]*tmplDataEncounter, len(inst.Preset.Encounter)),
			}
			partData.jobsMap[td.Spec] = jobData
		}

		jobData.Korea.Allstar = resp.AllStars[0].Points
		jobData.Korea.Rank = resp.AllStars[0].Rank
		jobData.Korea.RankPercent = resp.AllStars[0].RankPercent

		for _, respEnc := range resp.Rankings {
			if respEnc.Allstar == nil {
				continue
			}

			encID := respEnc.Encounter.ID

			encData, ok := jobData.encountersMap[encID]
			if !ok {
				encData = &tmplDataEncounter{
					EncounterID:   encID,
					EncounterName: inst.Preset.EncounterMap[encID].Name,
				}
				jobData.encountersMap[encID] = encData
			}

			encData.Kills = respEnc.TotalKills

			encData.EncounterName = inst.Preset.EncounterMap[encID].Name
			encData.Korea.Allstar = respEnc.Allstar.Points

			if respEnc.Allstar.Rank.Ok {
				encData.Korea.Rank = respEnc.Allstar.Rank.V
				encData.Korea.RankPercent = respEnc.Allstar.RankPercent.V
			}

		}
	}

	progressPercent := func() float32 {
		return float32(atomic.LoadInt32(&worked)) / float32(len(todoList)) * 100
	}
	progress := func() {
		p := progressPercent()
		log.Printf("UpdateKrEncounterRank %s@%s (%.2f %%)\n", inst.CharName, inst.CharServer, p)
		inst.progress("[2 / 3] 세부 등수 불러오는 중 %.2f %%", p)
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	var respCache RespZoneRanking
	for _, todo := range todoList {
		ok := cacheEncounterRank(
			inst.CharName,
			inst.CharServer,
			todo.ZoneID,
			todo.Partition,
			todo.Spec,
			&respCache,
			false,
		)
		if ok {
			do(todo.Key, &respCache, false)
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
					CharacterData struct {
						Character map[string]*RespZoneRanking `json:"character"`
					} `json:"characterData"`
				} `json:"data"`
			}

			q := struct {
				CharName   string
				CharServer string
				Data       []*TodoData
			}{
				CharName:   inst.CharName,
				CharServer: inst.CharServer,
				Data:       query,
			}

			err := analysis.CallGraphQL(ctx, tmplEncounterRank, &q, &resp)
			if err != nil {
				sentry.CaptureException(err)
				fmt.Printf("%+v\n", errors.WithStack(err))
				return err
			}

			for hash, zoneData := range resp.Data.CharacterData.Character {
				do(hash, zoneData, true)
			}
			progress()

			return nil
		}
	}

	query := make([]*TodoData, 0, maxAllstar)
	for {
		pp.Reset(inst.ctx)

		qCount := 0
		for _, todo := range todoList {
			if todo.retries < 3 && !todo.done {
				todo.retries++
				query = append(query, todo)

				if len(query) == maxAllstar {
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
				"%s@%s / ZoneID: %d / Partition: %d / Spec: %s",
				inst.CharName, inst.CharServer,
				todo.ZoneID, todo.Partition, todo.Spec,
			))
			return false
		}
	}

	progress()

	return true
}
