package analysis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"ffxiv_check/ffxiv"
	"ffxiv_check/share"

	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
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

	charState string

	Reports map[string]*analysisReport
	Fights  map[fightKey]*analysisFight

	encounterNames map[int]string

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

var (
	reEnc = regexp.MustCompile(`^e[^_]*_?(\d+)$`)

	sbPool = sync.Pool{
		New: func() interface{} {
			sb := new(strings.Builder)
			sb.Grow(16 * 1024)
			sb.Reset()
			return sb
		},
	}
	bufPool = sync.Pool{
		New: func() interface{} {
			buf := new(bytes.Buffer)
			buf.Grow(16 * 1024)
			buf.Reset()
			return buf
		},
	}
)

func (inst *analysisInstance) try(f func() error) (err error) {
	for i := 0; i < maxRetries; i++ {
		err = f()

		if err == nil {
			break
		}
		if share.IsContextClosedError(err) {
			return err
		}
		if i+1 < maxRetries {
			time.Sleep(3 * time.Second)
		}
	}
	return err
}

func (inst *analysisInstance) progress(format string, args ...interface{}) {
	inst.progressString <- fmt.Sprintf(format, args...)
}

func (inst *analysisInstance) callGraphQl(ctx context.Context, tmpl *template.Template, tmplData interface{}, respData interface{}) error {
	sb := sbPool.Get().(*strings.Builder)
	defer sbPool.Put(sb)

	sb.Reset()
	err := tmpl.Execute(sb, tmplData)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return err
	}

	queryData := struct {
		Query string `json:"query"`
	}{
		Query: sb.String(),
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)

	buf.Reset()
	err = jsoniter.NewEncoder(buf).Encode(&queryData)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return err
	}

	if ctx == nil {
		ctx = inst.ctx
	}

	req, ok := client.NewRequest(
		ctx,
		"POST",
		"https://ko.fflogs.com/api/v2/client",
		buf,
	)
	if !ok {
		return err
	}

	req.Header.Set("Content-Type", "application/json; encoding=utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if !share.IsContextClosedError(err) {
			sentry.CaptureException(err)
			fmt.Printf("%+v\n", errors.WithStack(err))
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		client.Reset()
	}

	err = jsoniter.NewDecoder(resp.Body).Decode(&respData)
	if err != io.EOF && err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return err
	}

	return nil
}
