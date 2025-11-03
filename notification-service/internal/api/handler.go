package api

import (
	"net/http"

	"notification-service/internal/processor"
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
