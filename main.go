package main

import (
	"ffxiv_check/frontend"

	"github.com/gin-gonic/gin"
)

func main() {
	g := gin.New()

	frontend.Route(g)

	g.Run("127.0.0.1:5555")
}
