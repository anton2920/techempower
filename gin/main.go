package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func PlaintextHandler(c *gin.Context) {
	c.String(200, "Hello, World!")
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	serverHeader := []string{"Gin"}
	r.Use(func(c *gin.Context) {
		c.Writer.Header()["Server"] = serverHeader
	})
	r.GET("/plaintext", PlaintextHandler)

	log.Printf("Listening on 0.0.0.0:7073...")
	r.Run("0.0.0.0:7073")
}
