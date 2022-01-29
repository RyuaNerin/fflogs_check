package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	websockEmptyClosure = websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
)

func init() {
	recaptcha.Init(os.Getenv("GOOGLE_RECAPTCHA_V3_SECRET"))
}

func routeRequest(c *gin.Context) {
	ctx, ctxCancel := context.WithCancel(c.Request.Context())
	defer ctxCancel()

	ws, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		panic(err)
	}
	defer ws.Close()

	////////////////////////////////////////////////////////////////////////////////////////////////////

	ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		fmt.Printf("%+v\n", errors.WithStack(err))
		return
	}

	var remoteAddr string
	if v := c.GetHeader("X-Forwarded-For"); v != "" {
		remoteAddr = v
	}
	if remoteAddr == "" {
		if v := c.GetHeader("X-Real-Ip"); v != "" {
			remoteAddr = v
		}
	}
	if remoteAddr == "" {
		remoteAddr = c.Request.RemoteAddr
		if idx := strings.IndexByte(remoteAddr, ':'); idx >= 0 {
			remoteAddr = remoteAddr[:idx]
		}
	}

	ok, err := recaptcha.Confirm(remoteAddr, string(msg))
	if err != nil || !ok {
		return
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////

	done := make(chan struct{}, 1)

	q := queueData{
		conn:    ws,
		context: ctx,
		done:    done,
	}
	q.Ready()

	err = ws.ReadJSON(&q.opt)
	if err != nil {
		panic(err)
	}

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
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				q.lock.Lock()
				err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
				q.lock.Unlock()
				if err != nil {
					ctxCancel()
					return
				}
			case <-done:
				return
			}
		}
	}()

	q.opt.Context = ctx
	q.opt.Progress = q.Progress

	queueLock.Lock()
	q.Reorder(len(queue))
	if len(queue) == 0 {
		queueWake <- struct{}{}
	}
	queue = append(queue, &q)
	queueLock.Unlock()

	select {
	case <-q.done:
		log.Println("sent")

		q.lock.Lock()
		err = ws.WriteMessage(websocket.CloseMessage, websockEmptyClosure)
		q.lock.Unlock()
		if err != nil {
			log.Printf("%+v\n", errors.WithStack(err))
		}
		select {
		case <-time.After(time.Second * 10):
		case <-ctx.Done():
		}

		log.Println("ok")

	case <-ctx.Done():
		log.Println("done")

		q.lock.Lock()
		q.skip = true
		q.lock.Unlock()
	}
}
