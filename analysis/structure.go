package analysis

type CharacterRanking struct {
	Spec     string `json:"spec"`
	ReportID string `json:"reportID"`
	FightID  int    `json:"fightID"`

	EncounterID   int    `json:"encounterID"`
	EncounterName string `json:"encounterName"`
}

//////////////////////////////////////////////////

type Report struct {
	Fights []struct {
		ID        int `json:"id"`
		StartTime int `json:"start_time"`
		EndTime   int `json:"end_time"`
	} `json:"fights"`
	Friendlies []struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Server *string `json:"server"`
		Job    string  `json:"type"`
		Fights []struct {
			ID int `json:"id"`
		} `json:"fights"`
	} `json:"friendlies"`
}

//////////////////////////////////////////////////

type Events struct {
	Count  int `json:"count"`
	Events []struct {
		Timestamp int    `json:"timestamp"`
		Type      string `json:"type"`
		Ability   struct {
			GUID int `json:"guid"`
			Type int `json:"type"`
		} `json:"ability"`
	} `json:"events"`
	NextPageTimestamp *int `json:"nextPageTimestamp"`
}
