package storage

import (
	"context"
	"notification-service/pkg/models"
	"time"
)

type NotificationStore interface {
	SaveNotification(ctx context.Context, notif models.Notification) error
	GetNotification(ctx context.Context, id string) (*models.Notification, error)
	UpdateNotificationStatus(ctx context.Context, id string, status string, errMsg string) error
	IncrementAttempts(ctx context.Context, id string, lastError string) (int, error)
	ScheduleRetry(ctx context.Context, notifID string, nextRetry time.Time, lastErr string) error
	GetDueRetries(ctx context.Context, before time.Time, limit int) ([]string, error)
	RemoveFromRetryQueue(ctx context.Context, notifID string) error
	UpdateAPIResponse(ctx context.Context, id string, statusCode int, body string) error // NEW
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}
