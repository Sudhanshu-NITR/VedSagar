package redisstore

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"notification-service/pkg/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

// Config holds the Redis connection settings.
type Config struct {
	Addr     string
	Username string
	Password string
	UseTLS   bool
}

// NewRedisStore creates and validates a Redis client using explicit config.
func NewRedisStore(ctx context.Context, cfg Config) (*RedisStore, error) {
	if cfg.Addr == "" {
		return nil, errors.New("redis address is empty")
	}

	opts := &redis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           0,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		MaxRetries:   3,
	}

	// Optional TLS for Redis Cloud or rediss://
	if cfg.UseTLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	rdb := redis.NewClient(opts)

	fmt.Print("✅ Redis connected successfully")

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisStore{rdb: rdb}, nil
}

func (s *RedisStore) Close(ctx context.Context) error {
	return s.rdb.Close()
}

func (s *RedisStore) notifKey(id string) string {
	return "notification:" + id
}

// SaveNotification stores/updates the notification hash
func (s *RedisStore) SaveNotification(ctx context.Context, notif models.Notification) error {
	key := s.notifKey(notif.ID)
	fields := map[string]interface{}{
		"event_id":   notif.EventID,
		"recipient":  notif.Recipient,
		"channel":    notif.Channel,
		"message":    notif.Message,
		"status":     notif.Status,
		"error":      notif.Error,
		"created_at": notif.Timestamp.Unix(),
		"updated_at": notif.Timestamp.Unix(),
	}
	if _, err := s.rdb.HSet(ctx, key, fields).Result(); err != nil {
		return fmt.Errorf("hset: %w", err)
	}
	return nil
}

// UpdateNotificationStatus marks final state for the attempt
func (s *RedisStore) UpdateNotificationStatus(ctx context.Context, id string, status string, errMsg string) error {
	key := s.notifKey(id)
	fields := map[string]interface{}{
		"status":     status,
		"error":      errMsg,
		"updated_at": time.Now().Unix(),
	}
	if _, err := s.rdb.HSet(ctx, key, fields).Result(); err != nil {
		return fmt.Errorf("update hset: %w", err)
	}
	return nil
}

const retryZSet = "retry_queue"

// ScheduleRetry writes last error and adds the ID to a time-ordered ZSET
func (s *RedisStore) ScheduleRetry(ctx context.Context, notifID string, nextRetry time.Time, lastErr string) error {
	if notifID == "" {
		return errors.New("sched retry: empty notifID")
	}

	if _, err := s.rdb.HSet(ctx, s.notifKey(notifID),
		"error", lastErr,
		"updated_at", time.Now().Unix(),
	).Result(); err != nil {
		return fmt.Errorf("schedule retry: hset metadata: %w", err)
	}

	score := float64(nextRetry.Unix())
	if _, err := s.rdb.ZAdd(ctx, retryZSet, redis.Z{
		Score:  score,
		Member: notifID,
	}).Result(); err != nil {
		return fmt.Errorf("schedule retry: zadd: %w", err)
	}
	return nil
}

// GetDueRetries fetches IDs with score (time) <= before
func (s *RedisStore) GetDueRetries(ctx context.Context, before time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}
	ids, err := s.rdb.ZRangeByScore(ctx, retryZSet, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%d", before.Unix()),
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("get due retries: %w", err)
	}
	return ids, nil
}

// RemoveFromRetryQueue removes an ID after success or max attempts
func (s *RedisStore) RemoveFromRetryQueue(ctx context.Context, notifID string) error {
	if notifID == "" {
		return errors.New("remove retry: empty notifID")
	}
	if _, err := s.rdb.ZRem(ctx, retryZSet, notifID).Result(); err != nil {
		return fmt.Errorf("zrem: %w", err)
	}
	return nil
}

func (s *RedisStore) GetNotification(ctx context.Context, id string) (*models.Notification, error) {
	notif := &models.Notification{}

	key := s.notifKey(id)
	result, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall: %w", err)
	}
	if len(result) == 0 {
		return nil, errors.New("notification not found")
	}

	notif.ID = id
	notif.EventID = result["event_id"]
	notif.Recipient = result["recipient"]
	notif.Channel = result["channel"]
	notif.Message = result["message"]
	notif.Status = result["status"]
	notif.Error = result["error"]

	if ts, ok := result["created_at"]; ok {
		if t, err := strconv.ParseInt(ts, 10, 64); err == nil {
			notif.Timestamp = time.Unix(t, 0)
		}
	}

	return notif, nil
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.rdb.Ping(ctx).Err()
}
