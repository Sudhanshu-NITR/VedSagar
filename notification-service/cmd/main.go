package main

import (
	"context"
	"log"
	"notification-service/internal/api"
	"notification-service/internal/processor"
	redisstore "notification-service/internal/storage/redis"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Update config usage to full Redis URL (future update)
	store, err := redisstore.NewRedisStore(ctx, redisstore.Config{
		Addr:     os.Getenv("REDIS_ADDR"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		UseTLS:   false,
	})
	if err != nil {
		log.Fatalf("redis init: %v", err)
	}
	defer store.Close(ctx)

	// Initialize dispatcher in processor package
	processor.Init(store)

	// Start retry worker with 1-min polls
	processor.StartRetryWorker(ctx, store, processor.Disp(), time.Minute)

	r := gin.Default()
	r.POST("/events", api.HandleEvent)
	r.GET("/health", api.HealthCheckHandler(store))

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
