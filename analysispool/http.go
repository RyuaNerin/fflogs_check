package analysispool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"ffxiv_check/analysis"

	"github.com/getsentry/sentry-go"
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
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return
	}

	q := queueData{
		ws:        ws,
		ctx:       ctx,
		ctxCancel: ctxCancel,
		chanResp:  make(chan *analysis.Statistic),
	}

	err = ws.ReadJSON(&q.opt)
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

	if !checkOptionValidation(&q.opt) {
		stat := analysis.Statistic{
			CharName:   q.opt.CharName,
			CharServer: q.opt.CharServer,
			CharRegion: q.opt.CharRegion,
			UpdatedAt:  time.Now().Format("2006-01-02 15:04:05"),
			State:      analysis.StatisticStateInvalid,
		}
		q.Succ(&stat)
		return
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	h := getOptionHash(&q.opt)

	var stat *analysis.Statistic
	if csStatistics.Load(h, &stat) {
		q.Succ(stat)
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
		q.Reorder(len(queue))
		queueLock.Unlock()

		stat = <-q.chanResp

		if stat == nil {
			q.Error()
		} else {
			csStatistics.Save(h, stat)
			q.Succ(stat)
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
