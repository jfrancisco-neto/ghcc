package main

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	logger := slog.Default()
	logger.Info("Webservice starging")

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.Run()
}
