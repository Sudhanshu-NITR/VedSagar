package channels

import (
	"context"
	"fmt"
	"time"

	"notification-service/internal/logger"
	"notification-service/pkg/models"
)

type EmailHandler struct{}

func (h *EmailHandler) Send(ctx context.Context, notif models.Notification) models.DispatchResult {
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return models.DispatchResult{
			NotificationID: notif.ID,
			Success:        false,
			Error:          "context cancelled",
			Timestamp:      time.Now(),
		}
	}

	logger.Info(fmt.Sprintf("[EMAIL] Sent to %s: %s", notif.Recipient, notif.Message))

	return models.DispatchResult{
		NotificationID: notif.ID,
		Success:        true,
		Error:          "",
		Timestamp:      time.Now(),
	}
}
