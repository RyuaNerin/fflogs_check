package analysis

type Statistics struct {
	CharName   string `json:"char_name"`
	CharServer string `json:"char_server"`
	CharRegion string `json:"char_region"`

	EncounterId int `json:"encounter_id"`

	// 각 직업군별로...
	Jobs    []*BuffUsageWithJob `json:"data"`
	jobsMap map[string]*BuffUsageWithJob
}

type BuffUsageWithJob struct {
	Job string `json:"job"`

	TotalKills int `json:"kills"` // 전체 킬 수

	Data    []*BuffUsage `json:"usage"`
	dataMap map[int]*BuffUsage
}

type BuffUsage struct {
	Skill BuffSkillInfo `json:"skill"`

	Usage    BuffStatistics `json:"usage"`    // 사용 횟수
	Cooldown BuffStatistics `json:"cooldown"` //쿨타임이였던 시간
}
type BuffSkillInfo struct {
	ID       int    `json:"skill_id"`
	Name     string `json:"skill_name"`
	Cooldown int
}
type BuffStatistics struct {
	data []float32
	Avg  float32 `json:"avg"`
	Med  float32 `json:"med"`
}
