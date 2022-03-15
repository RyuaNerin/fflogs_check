package perfection

const (
	statisticStateNormal   = "normal"
	statisticStateHidden   = "hidden"
	statisticStateNotFound = "notfound"
	statisticStateInvalid  = "invalid"
	statisticStateNoLog    = "nolog"
)

type statistic struct {
	UpdatedAt string `json:"updated_at"`

	State string `json:"state"`

	CharName   string `json:"char_name"`
	CharServer string `json:"char_server"`
	CharRegion string `json:"char_region"`

	FFLogsLink string `json:"fflogs_link"`

	Jobs    []*statisticJob          `json:"jobs"`
	jobsMap map[string]*statisticJob `json:"-"`

	Encounters    []*statisticEncounter       `json:"encounters"`
	encountersMap map[int]*statisticEncounter `json:"-"`
}

type statisticJob struct {
	ID  int    `json:"id"`
	Job string `json:"job"`

	Kills      int     `json:"kills"`
	Score      float32 `json:"score"`
	scoreSum   float64 `json:"-"`
	scoreCount int     `json:"-"`
}

type statisticEncounter struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	Kills      int     `json:"kills"`
	Score      float32 `json:"score"`
	scoreSum   float64 `json:"-"`
	scoreCount int     `json:"-"`

	Jobs    []*statisticEncounterJob          `json:"jobs"`
	jobsMap map[string]*statisticEncounterJob `json:"-"`
}

type statisticEncounterJob struct {
	ID  int    `json:"id"`
	Job string `json:"job"`

	Rank statisticRank `json:"rank"`

	Kills      int     `json:"kills"`
	Score      float32 `json:"score"`
	scoreSum   float64 `json:"-"`
	scoreCount int     `json:"-"`

	Skills    []*statisticSkill       `json:"skills"`
	skillsMap map[int]*statisticSkill `json:"-"`
}

type statisticSkill struct {
	Info BuffSkillInfo `json:"info"`

	Usage statisticSkillUsage `json:"usage"` // 사용 횟수

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
type statisticSkillUsage struct {
	data []int   `json:"-"`
	Avg  float32 `json:"avg"`
	Med  int     `json:"med"`
}

type StatisticSkillCooldown struct {
	data []float32 `json:"-"`
	Avg  float32   `json:"avg"`
	Med  float32   `json:"med"`
}

type statisticRank struct {
	Dps float32 `json:"dps"`
	Hps float32 `json:"hps"`
}
