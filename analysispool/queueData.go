package analysispool

import (
	"bytes"
	"context"
	"log"
	"strings"
	"sync"

	"ffxiv_check/analysis"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var (
	queueLock sync.Mutex
	queue     = make([]*queueData, 0, 16)
	queueWake = make(chan struct{}, 1)
)

type queueData struct {
	opt     analysis.AnalyzeOptions // 설정
	context context.Context

	msg  chan *bytes.Buffer
	done chan struct{}
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
		resp, ok := analysis.Analyze(&q.opt)
		log.Printf("End: %s@%s", q.opt.CharName, q.opt.CharServer)
		if !ok {
			q.Error()
		} else {
			q.Succ(resp)
		}
		q.done <- struct{}{}
	}
}

func (q *queueData) Reorder(order int) {
	resp := struct {
		Event string `json:"event"`
		Data  int    `json:"data"`
	}{
		Event: "waiting",
		Data:  order,
	}

	buf := eventRespBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	err := jsoniter.NewEncoder(buf).Encode(&resp)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
		eventRespBufferPool.Put(buf)
		return
	}

	q.msg <- buf
}

func (q *queueData) Start() {
	buf := eventRespBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(eventStart)
	q.msg <- buf
}

func (q *queueData) Progress(s string) {
	resp := struct {
		Event string `json:"event"`
		Data  string `json:"data"`
	}{
		Event: "progress",
		Data:  s,
	}

	buf := eventRespBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	err := jsoniter.NewEncoder(buf).Encode(&resp)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
		eventRespBufferPool.Put(buf)
		return
	}

	q.msg <- buf
}

func (q *queueData) Error() {
	buf := eventRespBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(eventError)
	q.msg <- buf
}

func (q *queueData) Succ(r *analysis.Statistics) {
	sb := tmplAnalysisPool.Get().(*strings.Builder)
	defer tmplAnalysisPool.Put(sb)

	err := tmplAnalysis.Execute(sb, r)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))

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

	buf := eventRespBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	err = jsoniter.NewEncoder(buf).Encode(&resp)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
		eventRespBufferPool.Put(buf)
		return
	}

	q.msg <- buf
}
