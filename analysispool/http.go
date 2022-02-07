package analysispool

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
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
		sentry.CaptureException(err)
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
		sentry.CaptureException(err)
		return
	}
	go func() {
		for {
			_, r, err := ws.NextReader()
			if err != nil {
				sentry.CaptureException(err)
				return
			}

			_, err = io.Copy(io.Discard, r)
			if err != nil && err != io.EOF {
				sentry.CaptureException(err)
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
					sentry.CaptureException(err)
					ctxCancel()
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	queueLock.Lock()
	if len(queue) == 0 {
		select {
		case queueWake <- struct{}{}:
		default:
		}
	}
	queue = append(queue, &q)
	q.Reorder(len(queue))
	queueLock.Unlock()

	<-q.done

	time.Sleep(time.Second)

	err = ws.WriteMessage(websocket.CloseMessage, websockEmptyClosure)
	if err != nil && err != websocket.ErrCloseSent {
		sentry.CaptureException(err)
	}

	ws.Close()
}
