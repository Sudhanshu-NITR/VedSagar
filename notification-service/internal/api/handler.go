package api

import (
	"net/http"

	"notification-service/internal/processor"
	"notification-service/internal/storage"
	"notification-service/pkg/models"

	"github.com/gin-gonic/gin"
)

func HandleEvent(c *gin.Context) {
	var event models.Event

	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	if err := processor.ProcessEvent(event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Event received and dispatched successfully",
	})
}

func HealthCheckHandler(store storage.NotificationStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		err := store.Ping(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	}
}
