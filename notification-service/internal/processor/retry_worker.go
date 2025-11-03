package processor

import (
	"context"
	"notification-service/internal/dispatcher"
	"notification-service/internal/logger"
	"notification-service/internal/storage"
	"notification-service/pkg/models"
	"time"
)

func StartRetryWorker(ctx context.Context, store storage.NotificationStore, dispatcher *dispatcher.Dispatcher, pollInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("Retry worker exiting")
				return
			case <-ticker.C:
				now := time.Now()
				ids, err := store.GetDueRetries(ctx, now, 100)
				if err != nil {
					logger.Error(err)
					continue
				}
				for _, id := range ids {
					go func(notifID string) {
						notif, err := store.GetNotification(ctx, notifID)
						if err != nil {
							logger.Error(err)
							// Optionally remove from retry queue if corrupted
							_ = store.RemoveFromRetryQueue(ctx, notifID)
							return
						}
						// Dispatch notification again
						results := dispatcher.DispatchEvent(ctx, models.Event{
							ID:         notif.EventID,
							Recipients: []string{notif.Recipient},
							Channels:   []string{notif.Channel},
							Message:    notif.Message,
						})

						// Check dispatch result (consider the first only)
						if len(results) == 0 {
							logger.Info("No dispatch result for retry notification " + notifID)
							return
						}
						res := results[0]

						if res.Success {
							_ = store.UpdateNotificationStatus(ctx, notifID, "success", "")
							_ = store.RemoveFromRetryQueue(ctx, notifID)
							logger.Info("Retry success for notification " + notifID)
						} else {
							_ = store.UpdateNotificationStatus(ctx, notifID, "failed", res.Error)
							// Schedule next retry with exponential backoff (e.g., 5 min * 2)
							nextRetry := time.Now().Add(10 * time.Minute) // Adjust backoff as appropriate
							_ = store.ScheduleRetry(ctx, notifID, nextRetry, res.Error)
							logger.Info("Retry failed for notification " + notifID + " error: " + res.Error)
						}
					}(id)
				}
			}
		}
	}()
}
