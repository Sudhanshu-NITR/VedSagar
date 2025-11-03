package main

import (
	"notification-service/internal/api"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Route setup
	router.POST("/events", api.HandleEvent)

	// Start server
	router.Run(":8080") // default port 8080
}
