// internal/processor/processor.go
package processor

import (
	"context"
	"fmt"
	"time"

	"notification-service/internal/dispatcher"
	"notification-service/internal/logger"
	"notification-service/pkg/models"
)

var disp *dispatcher.Dispatcher

func init() {
	disp = dispatcher.NewDispatcher()
}

func ProcessEvent(event models.Event) error {
	logger.Info(fmt.Sprintf("Processing Event: %s (%s)", event.Title, event.Type))

	// Validation
	if event.ID == "" || event.Type == "" {
		return fmt.Errorf("missing required fields: id or type")
	}

	if len(event.Recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	if len(event.Channels) == 0 {
		return fmt.Errorf("no channels specified")
	}

	// Dispatch to channels with a 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results := disp.DispatchEvent(ctx, event)

	// Log summary
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	logger.Info(fmt.Sprintf("Dispatch complete: %d/%d successful", successCount, len(results)))
	return nil
}
