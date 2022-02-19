package analysispool

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"

	"ffxiv_check/analysis"
	"ffxiv_check/share"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var (
	queueLock sync.Mutex
	queue     = make([]*queueData, 0, 16)
	queueWake = make(chan struct{}, 1)
)

type options struct {
	CharName   string   `json:"char_name"`
	CharServer string   `json:"char_server"`
	CharRegion string   `json:"char_region"`
	Preset     string   `json:"preset"`
	Jobs       []string `json:"jobs"`
}

type queueData struct {
	reqOpt      options
	analysisOpt analysis.AnalyzeOptions

	ws        *websocket.Conn
	ctx       context.Context
	ctxCancel func()

	chanResp chan *analysis.Statistic

	msgLock sync.Mutex
}

var (
	eventRespBufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 16*1024))
		},
	}

	eventReady = []byte(`{"event":"ready"}`)
	eventStart = []byte(`{"event":"start"}`)
	eventError = []byte(`{"event":"error"}`)
)

func init() {
	go queueWorker()
}

func queueWorker() {
	var q *queueData

	for {
		q = nil

		queueLock.Lock()
		if len(queue) > 0 {
			q = queue[0]

			if len(queue) > 1 {
				for i := 1; i < len(queue); i++ {
					go queue[1].Reorder(i)
					queue[0] = queue[1]
				}
			}
			queue = queue[:len(queue)-1]
		}
		queueLock.Unlock()
		if q == nil {
			<-queueWake
			continue
		}

		log.Printf("Start: %s@%s", q.analysisOpt.CharName, q.analysisOpt.CharServer)
		q.Start()
		resp, ok := analysis.Analyze(
			q.ctx,
			q.Progress,
			&q.analysisOpt,
		)
		log.Printf("End: %s@%s", q.analysisOpt.CharName, q.analysisOpt.CharServer)
		if ok {
			q.chanResp <- resp
		} else {
			q.chanResp <- nil
		}
	}
}

func (q *queueData) MessageJson(resp interface{}) error {
	buf := eventRespBufferPool.Get().(*bytes.Buffer)
	defer eventRespBufferPool.Put(buf)

	buf.Reset()

	err := jsoniter.NewEncoder(buf).Encode(&resp)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return err
	}

	return q.MessageBytes(buf.Bytes())
}

func (q *queueData) MessageBytes(data []byte) error {
	q.msgLock.Lock()
	defer q.msgLock.Unlock()

	return q.ws.WriteMessage(websocket.TextMessage, data)
}

func (q *queueData) Reorder(order int) {
	resp := struct {
		Event string `json:"event"`
		Data  int    `json:"data"`
	}{
		Event: "waiting",
		Data:  order,
	}

	err := q.MessageJson(&resp)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		q.ctxCancel()
	}
}

func (q *queueData) Start() {
	err := q.MessageBytes(eventStart)
	if err != nil {
		if err != websocket.ErrCloseSent {
			sentry.CaptureException(err)
			fmt.Printf("%+v\n", errors.WithStack(err))
		}
		q.ctxCancel()
	}
}

func (q *queueData) Progress(s string) {
	resp := struct {
		Event string `json:"event"`
		Data  string `json:"data"`
	}{
		Event: "progress",
		Data:  s,
	}

	err := q.MessageJson(&resp)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		q.ctxCancel()
	}
}

func (q *queueData) Error() {
	err := q.MessageBytes(eventError)
	if err != nil {
		if err != websocket.ErrCloseSent {
			sentry.CaptureException(err)
			fmt.Printf("%+v\n", errors.WithStack(err))
		}
		q.ctxCancel()
	}
}

func (q *queueData) Succ(r *analysis.Statistic) {
	sb := tmplAnalysisPool.Get().(*bytes.Buffer)
	defer tmplAnalysisPool.Put(sb)
	sb.Reset()

	err := tmplAnalysis.Execute(sb, r)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))

		q.Error()
		return
	}

	resp := struct {
		Event string `json:"event"`
		Data  string `json:"data"`
	}{
		Event: "complete",
		Data:  share.B2s(sb.Bytes()),
	}

	err = q.MessageJson(resp)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))

		q.ctxCancel()
		return
	}
}
