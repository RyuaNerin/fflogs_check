package main

import (
	"net/http"

	_ "ffxiv_check/share"

	"github.com/gin-gonic/gin"
)

func main() {
	g := gin.New()

	g.Use(gin.ErrorLogger())
	g.Use(gin.Recovery())

	g.NoMethod(func(c *gin.Context) { c.Status(http.StatusNotFound) })
	g.NoRoute(func(c *gin.Context) { c.Status(http.StatusNotFound) })

	g.GET("/api/analysis", routeRequest)

	g.Run("127.0.0.1:57381")
}
