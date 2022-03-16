package allstar

const (
	statisticStateNormal   = "normal"
	statisticStateHidden   = "hidden"
	statisticStateNotFound = "notfound"
	statisticStateInvalid  = "invalid"
	statisticStateNoLog    = "nolog"
)

type tmplData struct {
	UpdatedAt string `json:"updated_at"`
	State     string `json:"state"`

	CharName   string `json:"char_name"`
	CharServer string `json:"char_server"`

	FFLogsLink string `json:"fflogs_link"`

	ShowAllstar bool   `json:"show_allstar"`
	ZoneName    string `json:"zone_name"`

	Partitions    []*tmplDataPartition       `json:"partitions"`
	partitionsMap map[int]*tmplDataPartition `json:"-"`
}

type tmplDataPartition struct {
	PartitionIDKorea  int    `json:"partition_id_korea"`
	PartitionIDGlobal int    `json:"partition_id_global"`
	PartitionName     string `json:"partition_name"`

	Jobs    []*tmplDataJob          `json:"jobs"`
	jobsMap map[string]*tmplDataJob `json:"-"`
}

type tmplDataJob struct {
	Job string `json:"job"`

	Best bool `json:"best"`

	TotalKills int `json:"total_kills"`

	Korea  tmplDataRank `json:"korea"`
	Global tmplDataRank `json:"global"`

	Encounters    []*tmplDataEncounter       `json:"encounter"`
	encountersMap map[int]*tmplDataEncounter `json:"-"`
}

type tmplDataEncounter struct {
	EncounterID   int    `json:"encounter_id"`
	EncounterName string `json:"encounter_name"`

	Kills int `json:"kills"`

	Rdps   float32      `json:"rdps"`
	RdpsP  float32      `json:"rdps_p"`
	Korea  tmplDataRank `json:"korea"`
	Global tmplDataRank `json:"global"`
}

type tmplDataRank struct {
	Allstar     float32 `json:"allstar"` // Encounter 에서는 사용 안함
	Rank        int     `json:"rank"`
	RankPercent float32 `json:"rank_percent"` // Rank 색 계산용
}
