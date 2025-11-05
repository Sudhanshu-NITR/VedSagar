package processor

import (
	"context"
	"fmt"
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
							_ = store.RemoveFromRetryQueue(ctx, notifID) // remove corrupted entry
							return
						}

						// build a temporary Event for single dispatch
						results := dispatcher.DispatchEvent(ctx, models.Event{
							ID:         notif.EventID,
							Recipients: []string{notif.Recipient},
							Channels:   []string{notif.Channel},
							Message:    notif.Message,
						})

						if len(results) == 0 {
							logger.Info("No dispatch result for retry notification " + notifID)
							return
						}
						res := results[0]

						if res.Success {
							_ = store.UpdateNotificationStatus(ctx, notifID, "success", "")
							_ = store.RemoveFromRetryQueue(ctx, notifID)
							logger.Info("Retry success for notification " + notifID)
							return
						}

						// failure: increment attempts and decide next action
						newAttempts, err := store.IncrementAttempts(ctx, notifID, res.Error)
						if err != nil {
							logger.Error(fmt.Errorf("increment attempts for retry %s: %w", notifID, err))
							// Do not infinite-loop: schedule a conservative retry
							nextRetry := time.Now().Add(10 * time.Minute)
							_ = store.ScheduleRetry(ctx, notifID, nextRetry, res.Error)
							return
						}

						// determine max retries (fallback to 5)
						maxRetries := notif.MaxRetries
						if maxRetries == 0 {
							maxRetries = 5
						}

						if newAttempts >= maxRetries {
							_ = store.UpdateNotificationStatus(ctx, notifID, "failed_permanent", res.Error)
							_ = store.RemoveFromRetryQueue(ctx, notifID)
							logger.Info("Marked permanent failure for notification " + notifID + " after attempts")
							return
						}

						// exponential backoff: base * 2^(attempts-1), capped
						baseSeconds := int64(300)  // 5 minutes
						maxBackoff := int64(86400) // 24 hours
						delay := baseSeconds << uint(newAttempts-1)
						if delay > maxBackoff {
							delay = maxBackoff
						}
						nextRetry := time.Now().Add(time.Duration(delay) * time.Second)

						_ = store.UpdateNotificationStatus(ctx, notifID, "failed", res.Error)
						if err := store.ScheduleRetry(ctx, notifID, nextRetry, res.Error); err != nil {
							logger.Error(fmt.Errorf("schedule retry failed for %s: %w", notifID, err))
						} else {
							logger.Info(fmt.Sprintf("Retry scheduled for %s at %s (attempt %d/%d)", notifID, nextRetry.Format(time.RFC3339), newAttempts, maxRetries))
						}
					}(id)
				}
			}
		}
	}()
}
