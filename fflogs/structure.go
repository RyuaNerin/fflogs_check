package fflogs

type FFLogsEncounterRankingsResponse struct {
	Data struct {
		WorldData struct {
			Encounter struct {
				Name string `json:"name"`
			} `json:"encounter"`
		} `json:"worldData"`
		CharacterData struct {
			Character map[string]struct {
				Ranks []struct {
					Report FFLogsEncounterRankingsReport `json:"report"`
				} `json:"ranks"`
			} `json:"character"`
		} `json:"characterData"`
	} `json:"data"`
}

type FFLogsEncounterRankingsReport struct {
	Code    string `json:"code"`
	FightID int    `json:"fightID"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type FFLogsReportSummaryResponse struct {
	Data struct {
		ReportData map[string]FFLogsReportResponse `json:"reportData"`
	} `json:"data"`
}

type FFLogsReportResponse struct {
	Fights []struct {
		StartTime int64 `json:"startTime"`
		EndTime   int64 `json:"endTime"`
	} `json:"fights"`
	PlayerDetails struct {
		Data struct {
			PlayerDetails struct {
				Dps     []FFLogsReportPlayerDetail `json:"dps"`
				Tanks   []FFLogsReportPlayerDetail `json:"tanks"`
				Healers []FFLogsReportPlayerDetail `json:"healers"`
			} `json:"playerDetails"`
		} `json:"data"`
	} `json:"playerDetails"`
}

type FFLogsReportPlayerDetail struct {
	Name   string `json:"name"`
	Id     int    `json:"id"`
	Server string `json:"server"`
	Type   string `json:"type"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type FFLogsReportCastsEventResponse struct {
	Data struct {
		ReportData map[string]FFLogsReportCastsEventData `json:"reportData"`
	} `json:"data"`
}

type FFLogsReportCastsEventData struct {
	Events struct {
		NextPageTimestamp int64                         `json:"nextPageTimestamp"`
		Data              []FFLogsReportCastsEventEntry `json:"data"`
	} `json:"events"`
}

type FFLogsReportCastsEventEntry struct {
	Timestamp     int64 `json:"timestamp"`
	AbilityGameID int   `json:"abilityGameID"`
}
