package redisstore

import (
	"context"
	"crypto/tls"
	"fmt"
	"notification-service/pkg/models"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(ctx context.Context, redisURL string) (*RedisStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	// Harden client timeouts and retries
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 2 * time.Second
	opts.WriteTimeout = 2 * time.Second
	opts.MaxRetries = 3
	opts.MinRetryBackoff = 100 * time.Millisecond
	opts.MaxRetryBackoff = 1 * time.Second

	if opts.TLSConfig == nil && isRediss(redisURL) {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	rdb := redis.NewClient(opts)

	// Fail fast if not reachable
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisStore{rdb: rdb}, nil
}

func isRediss(u string) bool {
	return len(u) >= 7 && (u[:7] == "rediss:" || u[:8] == "rediss://")
}

// Add near the top if not present
func (s *RedisStore) notifKey(id string) string { return "notification:" + id }

// SaveNotification stores/updates the notification hash
func (s *RedisStore) SaveNotification(ctx context.Context, notif models.Notification) error {
	key := s.notifKey(notif.ID)
	_, err := s.rdb.HSet(ctx, key, map[string]interface{}{
		"event_id":   notif.EventID,
		"recipient":  notif.Recipient,
		"channel":    notif.Channel,
		"message":    notif.Message,
		"status":     notif.Status,
		"error":      notif.Error,
		"created_at": notif.Timestamp.Unix(),
		"updated_at": notif.Timestamp.Unix(),
	}).Result()
	return err
}

// UpdateNotificationStatus marks final state for the attempt
func (s *RedisStore) UpdateNotificationStatus(ctx context.Context, id string, status string, errMsg string) error {
	key := s.notifKey(id)
	_, err := s.rdb.HSet(ctx, key, map[string]interface{}{
		"status":     status,
		"error":      errMsg,
		"updated_at": time.Now().Unix(),
	}).Result()
	return err
}

const retryZSet = "retry_queue"

// ScheduleRetry writes last error and adds the ID to a time-ordered ZSET
func (s *RedisStore) ScheduleRetry(ctx context.Context, notifID string, nextRetry time.Time, lastErr string) error {
	// Store last error for visibility
	if _, err := s.rdb.HSet(ctx, s.notifKey(notifID),
		"error", lastErr,
		"updated_at", time.Now().Unix(),
	).Result(); err != nil {
		return err
	}
	// Add to sorted set with next retry timestamp as score
	_, err := s.rdb.ZAdd(ctx, retryZSet, redis.Z{
		Score:  float64(nextRetry.Unix()),
		Member: notifID,
	}).Result()
	return err
}

// GetDueRetries fetches IDs with score (time) <= before
func (s *RedisStore) GetDueRetries(ctx context.Context, before time.Time, limit int) ([]string, error) {
	ids, err := s.rdb.ZRangeByScore(ctx, retryZSet, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%d", before.Unix()),
		Count: int64(limit),
	}).Result()
	return ids, err
}

// RemoveFromRetryQueue removes an ID after success or max attempts
func (s *RedisStore) RemoveFromRetryQueue(ctx context.Context, notifID string) error {
	_, err := s.rdb.ZRem(ctx, retryZSet, notifID).Result()
	return err
}
