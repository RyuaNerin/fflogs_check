package frontend

import (
	"context"
	"ffxiv_check/analysis"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	queueLock sync.Mutex
	queue     = make([]*queueData, 0, 16)
	queueWake = make(chan struct{}, 1)
)

type queueData struct {
	lock sync.Mutex

	conn *websocket.Conn

	opt     analysis.AnalyzeOptions
	context context.Context

	done chan struct{}

	skip bool
}

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
		resp, err := analysis.Analyze(&q.opt)
		if err != nil {
			log.Println(err)
			q.Error()
		} else {
			q.Succ(resp)
		}
		<-q.done
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
		log.Println(err)
	}
}

func (q *queueData) Start() {
	q.lock.Lock()
	defer q.lock.Unlock()

	resp := struct {
		Event string `json:"event"`
	}{
		Event: "start",
	}

	err := q.conn.WriteJSON(&resp)
	if err != nil {
		log.Println(err)
	}
}

func (q *queueData) Progress(p float32) {
	q.lock.Lock()
	defer q.lock.Unlock()

	resp := struct {
		Event string  `json:"event"`
		Data  float32 `json:"data"`
	}{
		Event: "progress",
		Data:  p,
	}

	err := q.conn.WriteJSON(&resp)
	if err != nil {
		log.Println(err)
	}
}

func (q *queueData) Error() {
	q.lock.Lock()
	defer q.lock.Unlock()

	resp := struct {
		Event string `json:"event"`
	}{
		Event: "error",
	}

	err := q.conn.WriteJSON(&resp)
	if err != nil {
		log.Println(err)
	}
}

func (q *queueData) Succ(r *analysis.Statistics) {
	q.lock.Lock()
	defer q.lock.Unlock()

	resp := struct {
		Event string               `json:"event"`
		Data  *analysis.Statistics `json:"data"`
	}{
		Event: "complete",
		Data:  r,
	}

	err := q.conn.WriteJSON(&resp)
	if err != nil {
		log.Println(err)
	}
}
