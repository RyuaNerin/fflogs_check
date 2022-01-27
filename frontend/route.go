package frontend

import (
	"net/http"

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
	g.Static("/static", dir+"static")

	g.Use(gin.ErrorLogger())
	g.Use(gin.Recovery())

	g.NoMethod(func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/") })
	g.NoRoute(func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/") })

	g.StaticFile("/", dir+"index.htm")
	g.GET("/analysis", routeRequest)
}
