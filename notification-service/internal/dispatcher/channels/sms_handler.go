package channels

import (
	"context"
	"fmt"
	"time"

	"notification-service/internal/logger"
	"notification-service/pkg/models"
)

type SMSHandler struct{}

func (h *SMSHandler) Send(ctx context.Context, notif models.Notification) models.DispatchResult {
	// Simulate API call delay
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return models.DispatchResult{
			NotificationID: notif.ID,
			Success:        false,
			Error:          "context cancelled",
			Timestamp:      time.Now(),
		}
	}

	// Simulate success/failure (90% success rate for demo)
	success := true // For now, always succeed
	logger.Info(fmt.Sprintf("[SMS] Sent to %s: %s", notif.Recipient, notif.Message))

	return models.DispatchResult{
		NotificationID: notif.ID,
		Success:        success,
		Error:          "",
		Timestamp:      time.Now(),
	}
}
