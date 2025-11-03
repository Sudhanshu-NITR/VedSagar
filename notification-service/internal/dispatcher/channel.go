// internal/dispatcher/channel.go
package dispatcher

import (
	"context"
	"notification-service/pkg/models"
)

// ChannelHandler defines the interface for all channel implementations
type ChannelHandler interface {
	Send(ctx context.Context, notif models.Notification) models.DispatchResult
}
