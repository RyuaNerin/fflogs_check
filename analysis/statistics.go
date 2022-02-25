package analysis

const (
	StatisticStateNormal   = "normal"
	StatisticStateHidden   = "hidden"
	StatisticStateNotFound = "notfound"
	StatisticStateInvalid  = "invalid"
	StatisticStateNoLog    = "nolog"
)

type Statistic struct {
	UpdatedAt string `json:"updated_at"`

	State string `json:"state"`

	CharName   string `json:"char_name"`
	CharServer string `json:"char_server"`
	CharRegion string `json:"char_region"`

	FFLogsLink string `json:"fflogs_link"`

	Jobs    []*StatisticJob          `json:"jobs"`
	jobsMap map[string]*StatisticJob `json:"-"`

	Encounters    []*StatisticEncounter       `json:"encounters"`
	encountersMap map[int]*StatisticEncounter `json:"-"`
}

type StatisticJob struct {
	ID  int    `json:"id"`
	Job string `json:"job"`

	Kills      int     `json:"kills"`
	Score      float32 `json:"score"`
	scoreSum   float64 `json:"-"`
	scoreCount int     `json:"-"`
}

type StatisticEncounter struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	Kills      int     `json:"kills"`
	Score      float32 `json:"score"`
	scoreSum   float64 `json:"-"`
	scoreCount int     `json:"-"`

	Jobs    []*StatisticEncounterJob          `json:"jobs"`
	jobsMap map[string]*StatisticEncounterJob `json:"-"`
}

type StatisticEncounterJob struct {
	ID  int    `json:"id"`
	Job string `json:"job"`

	Rank StatisticRank `json:"rank"`

	Kills      int     `json:"kills"`
	Score      float32 `json:"score"`
	scoreSum   float64 `json:"-"`
	scoreCount int     `json:"-"`

	Skills    []*StatisticSkill       `json:"skills"`
	skillsMap map[int]*StatisticSkill `json:"-"`
}

type StatisticSkill struct {
	Info BuffSkillInfo `json:"info"`

	Usage StatisticSkillUsage `json:"usage"` // 사용 횟수

	Cooldown StatisticSkillCooldown `json:"cooldown"` //쿨타임이였던 시간
}
type BuffSkillInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Cooldown int    `json:"cooldown"`
	Icon     string `json:"icon"`

	WithDowntime    bool `json:"downtime"`
	ContainsInScore bool `json:"contains_in_score"`
}
type StatisticSkillUsage struct {
	data []int   `json:"-"`
	Avg  float32 `json:"avg"`
	Med  int     `json:"med"`
}

type StatisticSkillCooldown struct {
	data []float32 `json:"-"`
	Avg  float32   `json:"avg"`
	Med  float32   `json:"med"`
}

type StatisticRank struct {
	Dps float32 `json:"dps"`
	Hps float32 `json:"hps"`
}
