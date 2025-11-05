package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"notification-service/internal/dispatcher/channels"
	"notification-service/internal/logger"
	"notification-service/internal/storage"
	"notification-service/pkg/models"
)

type Dispatcher struct {
	handlers map[string]ChannelHandler
	store    storage.NotificationStore
}

func NewDispatcher(store storage.NotificationStore) *Dispatcher {
	return &Dispatcher{
		handlers: map[string]ChannelHandler{
			"sms":   channels.NewSMSHandler(),
			"email": &channels.EmailHandler{},
			"push":  &channels.PushHandler{},
		},
		store: store,
	}
}

func (d *Dispatcher) DispatchEvent(ctx context.Context, event models.Event) []models.DispatchResult {
	results := []models.DispatchResult{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, channel := range event.Channels {
		handler, exists := d.handlers[channel]
		if !exists {
			logger.Info(fmt.Sprintf("Unknown channel: %s, skipping", channel))
			continue
		}
		for _, recipient := range event.Recipients {
			wg.Add(1)
			go func(ch string, rec string) {
				defer wg.Done()

				notif := models.Notification{
					ID:        fmt.Sprintf("notif-%d", time.Now().UnixNano()),
					EventID:   event.ID,
					Recipient: rec,
					Channel:   ch,
					Message:   event.Message,
					Status:    "pending",
					Timestamp: time.Now(),
				}

				// Save initial record
				if err := d.store.SaveNotification(ctx, notif); err != nil {
					logger.Error(fmt.Errorf("save notification: %w", err))
				}

				result := handler.Send(ctx, notif)

				if result.Success {
					_ = d.store.UpdateNotificationStatus(ctx, notif.ID, "success", "")
					logger.Info(fmt.Sprintf("✓ Dispatch success: %s to %s via %s", event.Title, rec, ch))
				} else {
					// increment attempts and persist last error
					newAttempts, err := d.store.IncrementAttempts(ctx, notif.ID, result.Error)
					if err != nil {
						logger.Error(fmt.Errorf("increment attempts: %w", err))
						// fallback: still schedule a retry to avoid losing it
						newAttempts = 1
					}

					// determine max retries (use stored value if present, else fallback)
					maxRetries := notif.MaxRetries
					if maxRetries == 0 {
						maxRetries = 5 // default if not set via model/env
					}

					// if we've hit or exceeded max retries, mark permanent failure
					if newAttempts >= maxRetries {
						_ = d.store.UpdateNotificationStatus(ctx, notif.ID, "failed_permanent", result.Error)
						_ = d.store.RemoveFromRetryQueue(ctx, notif.ID)
						logger.Info(fmt.Sprintf("✗ Permanent failure: %s to %s via %s - %s", event.Title, rec, ch, result.Error))
					} else {
						// compute exponential backoff
						baseSeconds := int64(300)  // 5 minutes base
						maxBackoff := int64(86400) // cap backoff at 24 hours
						delay := baseSeconds << uint(newAttempts-1)
						if delay > maxBackoff {
							delay = maxBackoff
						}
						nextRetry := time.Now().Add(time.Duration(delay) * time.Second)

						_ = d.store.UpdateNotificationStatus(ctx, notif.ID, "failed", result.Error)
						_ = d.store.ScheduleRetry(ctx, notif.ID, nextRetry, result.Error)
						logger.Info(fmt.Sprintf("✗ Dispatch failed: %s to %s via %s - %s (attempt %d/%d) will retry at %s",
							event.Title, rec, ch, result.Error, newAttempts, maxRetries, nextRetry.Format(time.RFC3339)))
					}
				}

				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}(channel, recipient)
		}
	}

	wg.Wait()
	return results
}
