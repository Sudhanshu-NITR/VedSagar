// internal/dispatcher/dispatcher.go
package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"notification-service/internal/dispatcher/channels"
	"notification-service/internal/logger"
	"notification-service/pkg/models"
)

type Dispatcher struct {
	handlers map[string]ChannelHandler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: map[string]ChannelHandler{
			"sms":   &channels.SMSHandler{},
			"email": &channels.EmailHandler{},
			"push":  &channels.PushHandler{},
		},
	}
}

// DispatchEvent sends notifications for all channels and recipients
func (d *Dispatcher) DispatchEvent(ctx context.Context, event models.Event) []models.DispatchResult {
	results := []models.DispatchResult{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// For each channel specified in the event
	for _, channel := range event.Channels {
		handler, exists := d.handlers[channel]
		if !exists {
			logger.Info(fmt.Sprintf("Unknown channel: %s, skipping", channel))
			continue
		}

		// For each recipient
		for _, recipient := range event.Recipients {
			wg.Add(1)

			// Spawn goroutine for concurrent dispatch
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

				result := handler.Send(ctx, notif)

				mu.Lock()
				results = append(results, result)
				mu.Unlock()

				if result.Success {
					logger.Info(fmt.Sprintf("✓ Dispatch success: %s to %s via %s", event.Title, rec, ch))
				} else {
					logger.Info(fmt.Sprintf("✗ Dispatch failed: %s to %s via %s - %s", event.Title, rec, ch, result.Error))
				}
			}(channel, recipient)
		}
	}

	wg.Wait()
	return results
}
