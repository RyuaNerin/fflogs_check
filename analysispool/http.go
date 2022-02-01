package analysispool

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	websockEmptyClosure = websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
)

func Do(ctx context.Context, ws *websocket.Conn) {
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	err := ws.WriteMessage(websocket.TextMessage, eventReady)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
		return
	}

	q := queueData{
		context: ctx,
		msg:     make(chan *bytes.Buffer),
		done:    make(chan struct{}),
	}

	err = ws.ReadJSON(&q.opt)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
		return
	}
	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	q.opt.Context = ctx
	q.opt.Progress = q.Progress

	passed := false
	switch {
	case len(q.opt.CharName) < 3:
	case len(q.opt.CharServer) < 3:
	case len(q.opt.CharRegion) < 2:
	case len(q.opt.Encouters) == 0:
	case len(q.opt.Jobs) == 0:
	default:
		passed = true
	}
	if !passed {
		q.Error()
		return
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	writeDone := make(chan struct{})
	go func() {
		defer func() {
			writeDone <- struct{}{}
		}()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-q.msg:
				if !ok {
					return
				} else {
					err := ws.WriteMessage(websocket.TextMessage, msg.Bytes())
					eventRespBufferPool.Put(msg)
					if err != nil {
						log.Printf("%+v\n", errors.WithStack(err))
						ctxCancel()
						return
					}
				}

			case <-ticker.C:
				err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
				if err != nil {
					log.Printf("%+v\n", errors.WithStack(err))
					ctxCancel()
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	q.opt.Context = ctx
	q.opt.Progress = q.Progress

	queueLock.Lock()
	q.Reorder(len(queue))
	if len(queue) == 0 {
		select {
		case queueWake <- struct{}{}:
		default:
		}
	}
	queue = append(queue, &q)
	queueLock.Unlock()

	<-q.done
	close(q.msg)
	<-writeDone

	err = ws.WriteMessage(websocket.CloseMessage, websockEmptyClosure)
	if err != nil {
		log.Printf("%+v\n", errors.WithStack(err))
	}

	ws.Close()
}
