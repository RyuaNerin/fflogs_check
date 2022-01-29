package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"strings"
	"sync"

	"ffxiv_check/analysis"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	queueLock sync.Mutex
	queue     = make([]*queueData, 0, 16)
	queueWake = make(chan struct{}, 1)
)

var (
	tmplAnalysis     *template.Template
	tmplAnalysisPool = sync.Pool{
		New: func() interface{} {
			b := new(strings.Builder)
			b.Grow(64 * 1024)

			return b
		},
	}
	tmplFuncMap = template.FuncMap{
		"fn": func(value float64) string {
			return fmt.Sprintf("%.2f", value)
		},
	}
)

func init() {
	tmplAnalysis = template.Must(template.New("analysis.tmpl.htm").Funcs(tmplFuncMap).ParseFiles("./analysis.tmpl.htm"))
}

type queueData struct {
	lock sync.Mutex

	conn *websocket.Conn

	opt     analysis.AnalyzeOptions
	context context.Context

	done chan struct{}

	skip bool
}

var (
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

		q.lock.Lock()
		skip := q.skip
		q.lock.Unlock()

		if skip {
			continue
		}

		log.Printf("Start: %s@%s", q.opt.CharName, q.opt.CharServer)
		q.Start()
		resp, err := analysis.Analyze(&q.opt)
		log.Printf("End: %s@%s", q.opt.CharName, q.opt.CharServer)
		if err != nil {
			log.Printf("%+v\n", errors.WithStack(err))
			q.Error()
		} else {
			q.Succ(resp)
		}
		q.done <- struct{}{}
	}
}

func (q *queueData) Ready() {
	q.lock.Lock()
	defer q.lock.Unlock()

	err := q.conn.WriteMessage(websocket.TextMessage, eventReady)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
	}
}

func (q *queueData) Reorder(order int) {
	q.lock.Lock()
	defer q.lock.Unlock()

	resp := struct {
		Event string `json:"event"`
		Data  int    `json:"data"`
	}{
		Event: "waiting",
		Data:  order,
	}

	err := q.conn.WriteJSON(&resp)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
	}
}

func (q *queueData) Start() {
	q.lock.Lock()
	defer q.lock.Unlock()

	err := q.conn.WriteMessage(websocket.TextMessage, eventStart)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
	}
}

func (q *queueData) Progress(s string) {
	q.lock.Lock()
	defer q.lock.Unlock()

	resp := struct {
		Event string `json:"event"`
		Data  string `json:"data"`
	}{
		Event: "progress",
		Data:  s,
	}

	err := q.conn.WriteJSON(&resp)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
	}
}

func (q *queueData) Error() {
	q.lock.Lock()
	defer q.lock.Unlock()

	err := q.conn.WriteMessage(websocket.TextMessage, eventError)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
	}
}

func (q *queueData) Succ(r *analysis.Statistics) {
	q.lock.Lock()
	defer q.lock.Unlock()

	sb := tmplAnalysisPool.Get().(*strings.Builder)
	defer tmplAnalysisPool.Put(sb)

	err := tmplAnalysis.Execute(sb, r)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))

		err := q.conn.WriteMessage(websocket.TextMessage, eventError)
		if err != nil {
			log.Printf("%+v\n", errors.WithStack(err))
		}
		return
	}

	resp := struct {
		Event string `json:"event"`
		Data  string `json:"data"`
	}{
		Event: "complete",
		Data:  sb.String(),
	}
	err = q.conn.WriteJSON(&resp)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
	}
}
