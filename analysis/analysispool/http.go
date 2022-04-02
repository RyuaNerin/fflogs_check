package analysispool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	websockEmptyClosure = websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")

	bufPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 16*1024))
		},
	}
)

func Do(ctx context.Context, ws *websocket.Conn) {
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	err := ws.WriteMessage(websocket.TextMessage, eventReady)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return
	}

	q := queueData{
		ws:         ws,
		ctx:        ctx,
		ctxCancel:  ctxCancel,
		chanResult: make(chan bool, 1),
	}

	err = ws.ReadJSON(&q.reqData)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return
	}
	go func() {
		for {
			_, r, err := ws.NextReader()
			if err != nil {
				return
			}

			_, err = io.Copy(io.Discard, r)
			if err != nil && err != io.EOF {
				return
			}
		}
	}()

	q.buf = bufPool.Get().(*bytes.Buffer)
	q.buf.Reset()
	defer bufPool.Put(q.buf)

	h := q.reqData.Hash()
	if csTemplate.LoadRaw(h, q.buf) {
		q.Succ(q.buf)
	} else {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
					if err != nil {
						if err != websocket.ErrCloseSent {
							sentry.CaptureException(err)
							fmt.Printf("%+v\n", errors.WithStack(err))
						}
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
		queueCount := len(queue)
		queueLock.Unlock()

		q.Reorder(queueCount)

		select {
		case <-ctx.Done():
		case ok := <-q.chanResult:
			if ok {
				q.Succ(q.buf)
				csTemplate.SaveRaw(h, q.buf)
			} else {
				q.Error()
			}
		}
	}

	time.Sleep(time.Second)

	err = ws.WriteMessage(websocket.CloseMessage, websockEmptyClosure)
	if err != nil && err != websocket.ErrCloseSent {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
	}

	ws.Close()
}
