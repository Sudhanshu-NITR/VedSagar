package processor

import (
	"context"
	"fmt"
	"time"

	"notification-service/internal/dispatcher"
	"notification-service/internal/logger"
	"notification-service/internal/storage"
	"notification-service/pkg/models"
)

var (
	disp *dispatcher.Dispatcher
)

func Disp() *dispatcher.Dispatcher {
	return disp
}

func Init(store storage.NotificationStore) {
	disp = dispatcher.NewDispatcher(store)
}

func ProcessEvent(event models.Event) error {
	logger.Info(fmt.Sprintf("Processing Event: %s (%s)", event.Title, event.Type))

	if event.ID == "" || event.Type == "" {
		return fmt.Errorf("missing required fields: id or type")
	}
	if len(event.Recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}
	if len(event.Channels) == 0 {
		return fmt.Errorf("no channels specified")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = disp.DispatchEvent(ctx, event)
	return nil
}
