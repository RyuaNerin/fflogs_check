package perfection

import (
	"context"
	"fmt"

	"ffxiv_check/ffxiv"
)

const (
	maxRetries = 3
	workers    = 8

	maxSummary = 50
	maxEvents  = 25
)

type fightKey struct {
	ReportID string
	FightID  int
}

type analysisInstance struct {
	ctx context.Context

	InpCharName            string
	InpCharServer          string
	InpCharRegion          string
	InpAdditionalPartition []int
	InpCharJobs            map[string]bool
	InpEncounterIDs        []int
	InpDifficulty          int `json:"difficulty"`

	charState string

	charID int

	stat *statistic

	Reports map[string]*analysisReport
	Fights  map[fightKey]*analysisFight

	encounterNames map[int]string
	encounterRanks map[int]*analysisRank

	progressString chan string

	skillSets *ffxiv.SkillSets
}
type analysisReport struct {
	ReportID string

	Fights []*analysisFight
}
type analysisFight struct {
	ReportID  string
	FightID   int
	StartTime int
	EndTime   int

	DoneSummary bool
	DoneEvents  bool

	EncounterID int
	Job         string

	SourceID int

	Casts  []analysisEvent
	Buffs  []analysisBuff
	Deaths []analysisDeath

	Debuff analysisDebuffs

	AutoAttacks int

	skillData map[int]*analysisFightSkill
}
type analysisEvent struct {
	timestamp int
	gameID    int
}
type analysisBuff struct {
	timestamp int
	gameID    int
	removed   bool
}
type analysisDeath struct {
	timestamp int
}
type analysisDebuffs struct {
	ReduceDamange analysisDebuff
}
type analysisDebuff struct {
	count  int
	uptime int
}
type analysisFightSkill struct {
	Used           int
	UsedForPercent int // 쿨 공유하는거 최대 사용 횟수 맞추기 위한 수...
	MaxForPercent  int
}
type analysisRank struct {
	Dps map[string]*analysisRankData
	Hps map[string]*analysisRankData
}
type analysisRankData struct {
	Data []fflogsRankData
	Echo []fflogsRankData
}
type fflogsRankData struct {
	Rank   float32
	Amount float32
}

func (inst *analysisInstance) progress(format string, args ...interface{}) {
	select {
	case <-inst.ctx.Done():
	case inst.progressString <- fmt.Sprintf(format, args...):
	}
}
