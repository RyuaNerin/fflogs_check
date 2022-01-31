package analysis

type Statistics struct {
	CharName   string `json:"char_name"`
	CharServer string `json:"char_server"`
	CharRegion string `json:"char_region"`

	Encounter []*StatisticEncounter `json:"data"`
}

type StatisticEncounter struct {
	Encounter StatisticEncounterInfo   `json:"encounter"`
	Jobs      []*StatisticJob          `json:"data"`
	jobsMap   map[string]*StatisticJob `json:"-"`
}

type StatisticEncounterInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Zone int    `json:"zone"`
}

type StatisticJob struct {
	Job string `json:"job"`

	TotalKills int `json:"kills"` // 전체 킬 수

	Data    []*StatisticSkill       `json:"data"`
	dataMap map[int]*StatisticSkill `json:"-"`
}

type StatisticSkill struct {
	Info BuffSkillInfo `json:"info"`

	Usage    BuffStatistics `json:"usage"`    // 사용 횟수
	Cooldown BuffStatistics `json:"cooldown"` //쿨타임이였던 시간
}
type BuffSkillInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Cooldown int    `json:"cooldown"`
	Icon     string `json:"icon"`
}
type BuffStatistics struct {
	data []float64 `json:"-"`
	Avg  float64   `json:"avg"`
	Med  float64   `json:"med"`
}
