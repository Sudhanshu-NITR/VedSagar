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
			"sms":   &channels.SMSHandler{},
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

				// Update status
				if result.Success {
					_ = d.store.UpdateNotificationStatus(ctx, notif.ID, "success", "")
					logger.Info(fmt.Sprintf("✓ Dispatch success: %s to %s via %s", event.Title, rec, ch))
				} else {
					_ = d.store.UpdateNotificationStatus(ctx, notif.ID, "failed", result.Error)
					// Schedule a retry 5 minutes later for now (configurable later)
					_ = d.store.ScheduleRetry(ctx, notif.ID, time.Now().Add(5*time.Minute), result.Error)
					logger.Info(fmt.Sprintf("✗ Dispatch failed: %s to %s via %s - %s", event.Title, rec, ch, result.Error))
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
