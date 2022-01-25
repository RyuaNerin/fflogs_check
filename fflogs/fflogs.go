package fflogs

import (
	"fmt"
	"sort"

	"ffxiv_check/ffxiv"
	_ "ffxiv_check/share"
)

type BuffUsageStatisticsWithJob struct {
	CharName   string
	CharServer string

	EncounterId   int
	EncounterName string

	BuffUsageEachJob map[string]*BuffUsageWithJob // 각 직업군별로...
}

type BuffUsageWithJob struct {
	TotalKills int // 전체 킬 수
	Data       map[int]*BuffUsage
}

type BuffUsage struct {
	BuffId       int
	BuffName     string
	BuffCooldown int

	usage    []int   // 사용 횟수
	UsageSum int     // 전체 사용 횟수
	UsageAvg float64 // 사용 횟수 (평균)
	UsageMed int     // 사용 횟수 (중간값)

	cooldown    []float64 // 쿨타임이였던 시간
	CooldownAvg float64   // 쿨타임이였던 시간 (평균)
	CooldownMed float64   // 쿨타임이였던 시간 (중간값)
}

func GetBuffUsage(name string, server string, encounterId int, includeEcho bool) (*BuffUsageStatisticsWithJob, error) {
	inst := instancePool.Get().(*instance)
	inst.CharName = name
	inst.CharServer = server
	inst.EncounterId = encounterId
	inst.IncludeEcho = includeEcho

	err := inst.getReportCodes()
	if err != nil {
		return nil, err
	}

	data, err := inst.getReportData()

	inst.bufPostData.Reset()
	inst.bufQueryString.Reset()
	instancePool.Put(inst)

	return data, err
}

func (inst *instance) getReportCodes() error {
	var respData FFLogsEncounterRankingsResponse

	err := inst.callOAuthRequest(tmplEncounterRankings, inst, &respData)
	if err != nil {
		return err
	}

	inst.ReportDataInnerList = inst.ReportDataInnerList[:0]
	for _, v := range respData.Data.CharacterData.Character {
		for _, rank := range v.Ranks {
			inst.ReportDataInnerList = append(
				inst.ReportDataInnerList,
				reportDataInner{
					Report: rank.Report,
				},
			)
		}
	}

	inst.EncounterName = respData.Data.WorldData.Encounter.Name

	return nil
}

func (inst *instance) getReportData() (r *BuffUsageStatisticsWithJob, err error) {
	err = inst.getReportSummary()
	if err != nil {
		return nil, err
	}

	err = inst.getReportCastsEvents()
	if err != nil {
		return nil, err
	}

	return inst.getReport(), nil
}

func (inst *instance) getReportSummary() error {
	detailList := make([]FFLogsReportPlayerDetail, 0, 16)
	update := func(reportKey string, reportData FFLogsReportResponse, cached bool) {
		detailList = detailList[:0]
		detailList = append(detailList, reportData.PlayerDetails.Data.PlayerDetails.Dps...)
		detailList = append(detailList, reportData.PlayerDetails.Data.PlayerDetails.Tanks...)
		detailList = append(detailList, reportData.PlayerDetails.Data.PlayerDetails.Healers...)

		for i, reportInner := range inst.ReportDataInnerList {
			if fmt.Sprintf("_%s_%d", reportInner.Report.Code, reportInner.Report.FightID) == reportKey {
				for _, detail := range detailList {
					if detail.Name == inst.CharName && detail.Server == inst.CharServer {
						inst.ReportDataInnerList[i].SourceId = detail.Id
						inst.ReportDataInnerList[i].Job = detail.Type

						inst.ReportDataInnerList[i].FightStartTime = reportData.Fights[0].StartTime
						inst.ReportDataInnerList[i].FightEndTime = reportData.Fights[0].EndTime

						if !cached {
							reportInner.saveReportSummary(reportData)
						}
						break
					}
				}

				break
			}
		}
	}

	todo := len(inst.ReportDataInnerList)
	for i, reportInner := range inst.ReportDataInnerList {
		ok, reportData := reportInner.loadReportSummary()
		if ok {
			inst.ReportDataInnerList[i].Done = true
			todo--

			update(fmt.Sprintf("_%s_%d", reportInner.Report.Code, reportInner.Report.FightID), reportData, true)
		} else {
			inst.ReportDataInnerList[i].Done = false
		}
	}

	if todo > 0 {
		var respData FFLogsReportSummaryResponse

		err := inst.callOAuthRequest(tmplReportSummary, inst, &respData)
		if err != nil {
			return err
		}

		for reportCode, reportData := range respData.Data.ReportData {
			update(reportCode, reportData, false)
		}
	}

	return nil
}

func (inst *instance) getReportCastsEvents() error {
	update := func(reportKey string, castsTableData FFLogsReportCastsEventData, cached bool) bool {
		end := false

		for i, reportInner := range inst.ReportDataInnerList {
			if fmt.Sprintf("_%s_%d_%d", reportInner.Report.Code, reportInner.Report.FightID, reportInner.SourceId) == reportKey {
				inst.ReportDataInnerList[i].Events = append(inst.ReportDataInnerList[i].Events, castsTableData.Events.Data...)

				if !cached {
					reportInner.saveReportCastsEvents(castsTableData)
				}

				inst.ReportDataInnerList[i].EventsNextPage = castsTableData.Events.NextPageTimestamp
				if castsTableData.Events.NextPageTimestamp == 0 {
					end = true
				}

				break
			}
		}

		return end
	}

	todo := len(inst.ReportDataInnerList)
	for i, reportInner := range inst.ReportDataInnerList {
		ok, reportData := reportInner.loadReportCastsEvents()
		if ok {
			inst.ReportDataInnerList[i].Done = true
			todo--

			update(fmt.Sprintf("_%s_%d_%d", reportInner.Report.Code, reportInner.Report.FightID, reportInner.SourceId), reportData, true)
		} else {
			inst.ReportDataInnerList[i].Done = false
		}
	}

	for todo > 0 {
		var respData FFLogsReportCastsEventResponse

		err := inst.callOAuthRequest(tmplReportCastsEvents, inst, &respData)
		if err != nil {
			return err
		}

		for reportCode, reportData := range respData.Data.ReportData {
			if update(reportCode, reportData, false) {
				todo--
			}
		}
	}

	return nil
}

func (inst *instance) getReport() (r *BuffUsageStatisticsWithJob) {
	r = &BuffUsageStatisticsWithJob{
		CharName:         inst.CharName,
		CharServer:       inst.CharServer,
		EncounterId:      inst.EncounterId,
		EncounterName:    inst.EncounterName,
		BuffUsageEachJob: make(map[string]*BuffUsageWithJob),
	}

	for _, report := range inst.ReportDataInnerList {
		buffUsageMap, ok := r.BuffUsageEachJob[report.Job]
		if !ok {
			buffUsageMap = &BuffUsageWithJob{
				Data: make(map[int]*BuffUsage),
			}
			r.BuffUsageEachJob[report.Job] = buffUsageMap
		}
		buffUsageMap.TotalKills++

		fightTime := report.FightEndTime - report.FightStartTime

		for _, skillId := range ffxiv.SkillDataEachJob[report.Job] {
			skillInfo := ffxiv.SkillDataMap[skillId]

			buffUsage, ok := buffUsageMap.Data[skillId]
			if !ok {
				buffUsage = &BuffUsage{
					BuffId:       skillInfo.ID,
					BuffCooldown: skillInfo.Cooldown,
					BuffName:     skillInfo.Name,
				}
				buffUsageMap.Data[skillId] = buffUsage
			}

			used := 0
			nextCooldown := int64(0)
			totalCooldown := int64(0)

			for _, event := range report.Events {
				if event.AbilityGameID != skillId {
					continue
				}

				when := event.Timestamp - report.FightStartTime

				if skillInfo.Cooldown > 0 {
					totalCooldown += when - int64(nextCooldown)
					nextCooldown = when + int64(skillInfo.Cooldown)
				}

				used++
			}

			if nextCooldown > fightTime {
				totalCooldown = fightTime - nextCooldown
			}

			buffUsage.usage = append(buffUsage.usage, used)
			buffUsage.cooldown = append(buffUsage.cooldown, float64(totalCooldown)/float64(fightTime)*100.0)
		}
	}

	for _, d := range r.BuffUsageEachJob {
		for _, buffUsage := range d.Data {
			sort.Ints(buffUsage.usage)
			sort.Float64s(buffUsage.cooldown)

			var usageSum int = 0
			for _, u := range buffUsage.usage {
				usageSum += u
			}
			buffUsage.UsageMed = buffUsage.usage[len(buffUsage.usage)/2]
			buffUsage.UsageAvg = float64(usageSum) / float64(len(buffUsage.usage))

			////////////////////////////////////////////////////////////////////////////////////////////////////

			var cooldownSum float64 = 0
			for _, u := range buffUsage.cooldown {
				cooldownSum += u
			}
			buffUsage.CooldownMed = buffUsage.cooldown[len(buffUsage.cooldown)/2]
			buffUsage.CooldownAvg = cooldownSum / float64(len(buffUsage.usage))

			////////////////////////////////////////////////////////////////////////////////////////////////////

		}
	}

	return
}
