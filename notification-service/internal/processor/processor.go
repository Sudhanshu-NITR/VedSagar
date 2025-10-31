package processor

import (
	"fmt"
	"notification-service/internal/logger"
	"notification-service/pkg/models"
)

// ProcessEvent handles basic validation and logs the event
func ProcessEvent(event models.Event) error {
	// Basic validation
	if event.ID == "" || event.Type == "" {
		return fmt.Errorf("missing required fields: id or type")
	}

	// For now, just log it
	logger.Info(fmt.Sprintf("Received Event: %s (%s) - Severity: %s",
		event.Title, event.Type, event.Severity))

	return nil
}
