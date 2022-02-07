package analysispool

import (
	"bytes"
	"context"
	"log"
	"strings"
	"sync"

	"ffxiv_check/analysis"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

var (
	queueLock sync.Mutex
	queue     = make([]*queueData, 0, 16)
	queueWake = make(chan struct{}, 1)
)

type queueData struct {
	opt analysis.AnalyzeOptions // 설정

	ws        *websocket.Conn
	ctx       context.Context
	ctxCancel func()

	done chan struct{}

	msgLock sync.Mutex
}

var (
	eventRespBufferPool = sync.Pool{
		New: func() interface{} {
			b := new(bytes.Buffer)
			b.Grow(64 * 1024)

			return b
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

		log.Printf("Start: %s@%s", q.opt.CharName, q.opt.CharServer)
		q.Start()
		resp, ok := analysis.Analyze(
			q.ctx,
			q.Progress,
			&q.opt,
		)
		log.Printf("End: %s@%s", q.opt.CharName, q.opt.CharServer)
		if !ok {
			q.Error()
		} else {
			q.Succ(resp)
		}
		q.done <- struct{}{}
	}
}

func (q *queueData) MessageJson(resp interface{}) error {
	buf := eventRespBufferPool.Get().(*bytes.Buffer)
	defer eventRespBufferPool.Put(buf)

	buf.Reset()

	err := jsoniter.NewEncoder(buf).Encode(&resp)
	if err != nil {
		sentry.CaptureException(err)
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
		q.ctxCancel()
	}
}

func (q *queueData) Start() {
	err := q.MessageBytes(eventStart)
	if err != nil {
		sentry.CaptureException(err)
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
		q.ctxCancel()
	}
}

func (q *queueData) Error() {
	err := q.MessageBytes(eventError)
	if err != nil {
		if err != websocket.ErrCloseSent {
			sentry.CaptureException(err)
		}
		q.ctxCancel()
	}
}

func (q *queueData) Succ(r *analysis.Statistics) {
	sb := tmplAnalysisPool.Get().(*strings.Builder)
	defer tmplAnalysisPool.Put(sb)

	err := tmplAnalysis.Execute(sb, r)
	if err != nil {
		sentry.CaptureException(err)

		q.Error()
		return
	}

	resp := struct {
		Event string `json:"event"`
		Data  string `json:"data"`
	}{
		Event: "complete",
		Data:  sb.String(),
	}

	err = q.MessageJson(&resp)
	if err != nil {
		sentry.CaptureException(err)
		q.ctxCancel()
	}
}
