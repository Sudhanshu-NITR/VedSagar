package channels

import (
	"context"
	"fmt"
	"time"

	"notification-service/internal/logger"
	"notification-service/pkg/models"
)

type PushHandler struct{}

func (h *PushHandler) Send(ctx context.Context, notif models.Notification) models.DispatchResult {
	select {
	case <-time.After(120 * time.Millisecond):
	case <-ctx.Done():
		return models.DispatchResult{
			NotificationID: notif.ID,
			Success:        false,
			Error:          "context cancelled",
			Timestamp:      time.Now(),
		}
	}

	logger.Info(fmt.Sprintf("[PUSH] Sent to device %s: %s", notif.Recipient, notif.Message))

	return models.DispatchResult{
		NotificationID: notif.ID,
		Success:        true,
		Error:          "",
		Timestamp:      time.Now(),
	}
}
