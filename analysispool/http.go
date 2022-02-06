package analysispool

import (
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
		ws:        ws,
		ctx:       ctx,
		ctxCancel: ctxCancel,
		done:      make(chan struct{}),
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

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
				if err != nil {
					ctxCancel()
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

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

	time.Sleep(time.Second)

	err = ws.WriteMessage(websocket.CloseMessage, websockEmptyClosure)
	if err != nil && err != websocket.ErrCloseSent {
		log.Printf("%+v\n", errors.WithStack(err))
	}

	ws.Close()
}
