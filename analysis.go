package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ffxiv_check/analysispool"

	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func init() {
	recaptcha.Init(os.Getenv("GOOGLE_RECAPTCHA_V3_SECRET"))
}

func routeRequest(c *gin.Context) {
	ws, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		c.Status(http.StatusBadRequest)
		return
	}
	defer ws.Close()

	////////////////////////////////////////////////////////////////////////////////////////////////////

	ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		sentry.CaptureException(err)
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
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return
	}

	analysispool.Do(c.Request.Context(), ws)
}
