package frontend

import (
	"log"
	"net/http"

	"ffxiv_check/analysis"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	dir = "./frontend/public/"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func Route(g *gin.Engine) {
	g.LoadHTMLGlob(dir + "*.tmpl.htm")
	g.Static("/static", dir+"static")

	g.Use(gin.ErrorLogger())
	g.Use(gin.Recovery())

	g.NoMethod(func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/") })
	g.NoRoute(func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/") })

	g.StaticFile("/", dir+"index.htm")
	g.GET("/analysis", routeRequest)
}

func routeRequest(c *gin.Context) {
	ws, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		panic(err)
	}
	defer ws.Close()

	var opt analysis.AnalyzeOptions
	err = ws.ReadJSON(&opt)
	if err != nil {
		panic(err)
	}

	opt.Context = c.Request.Context()

	chDone := make(chan *analysis.Statistics)
	go func() {
		resp, err := analysis.Analyze(&opt)
		if err != nil {
			log.Println(err)
			chDone <- nil
			return
		}

		chDone <- resp
	}()

	var resp struct {
		Succeed bool                 `json:"status"`
		Data    *analysis.Statistics `json:"data"`
	}

	resp.Data = <-chDone
	resp.Succeed = resp.Data != nil

	err = ws.WriteJSON(<-chDone)
	if err != nil {
		panic(err)
	}
}
