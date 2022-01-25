package main

import "github.com/gin-gonic/gin"

func main() {
	g := gin.New()

	g.Run("127.0.0.1:5555")
}
