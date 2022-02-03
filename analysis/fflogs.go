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

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

const (
	maxRetries = 3
	workers    = 8

	maxSummary = 50
	maxEvents  = 50
)

type fightKey struct {
	ReportID string
	FightID  int
}

type analysisInstance struct {
	ctx context.Context

	CharName            string
	CharServer          string
	CharRegion          string
	AdditionalPartition []int
	CharJobs            map[string]bool
	EncounterIDs        []int

	Reports map[string]*analysisReport
	Fights  map[fightKey]*analysisFight

	encounterNames map[int]string

	progressString chan string
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

	EncounterID int
	Job         string

	SourceID int

	Events []analysisEvent
}

type analysisEvent struct {
	avilityID int
	timestamp int

	icon____ string
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
		if i+1 < maxRetries {
			time.Sleep(3 * time.Second)
		}
	}
	return err
}

func (inst *analysisInstance) progress(format string, args ...interface{}) {
	select {
	case inst.progressString <- fmt.Sprintf(format, args...):
	default:
	}
}

func (inst *analysisInstance) callGraphQl(ctx context.Context, tmpl *template.Template, tmplData interface{}, respData interface{}) error {
	sb := sbPool.Get().(*strings.Builder)
	defer sbPool.Put(sb)

	sb.Reset()
	err := tmpl.Execute(sb, tmplData)
	if err != nil {
		return errors.WithStack(err)
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
		return err
	}

	if ctx == nil {
		ctx = inst.ctx
	}

	req, err := client.NewRequest(
		ctx,
		"POST",
		"https://ko.fflogs.com/api/v2/client",
		buf,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json; encoding=utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = jsoniter.NewDecoder(resp.Body).Decode(&respData)
	if err != io.EOF && err != nil {
		return err
	}

	return nil
}
