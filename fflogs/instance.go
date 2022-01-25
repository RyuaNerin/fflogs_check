package fflogs

import (
	"bytes"
	"sync"
)

type reportDataInner struct {
	Report   FFLogsEncounterRankingsReport
	SourceId int
	Done     bool
	Job      string

	Events         []FFLogsReportCastsEventEntry
	EventsNextPage int64

	FightStartTime int64
	FightEndTime   int64
}

type instance struct {
	bufQueryString bytes.Buffer
	bufPostData    bytes.Buffer

	CharName   string
	CharServer string

	EncounterId   int
	EncounterName string

	IncludeEcho bool

	ReportDataInnerList []reportDataInner
}

var instancePool = sync.Pool{
	New: func() interface{} {
		inst := new(instance)
		inst.bufPostData.Grow(16 * 1024)
		inst.bufQueryString.Grow(16 * 1024)
		return inst
	},
}
