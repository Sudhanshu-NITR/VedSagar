package main

import (
	"context"
	"log"

	"notification-service/internal/api"
	"notification-service/internal/config"
	"notification-service/internal/processor"
	redisstore "notification-service/internal/storage/redis"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	// Build store with a startup connectivity check
	ctx, cancel := context.WithTimeout(context.Background(), 10_000_000_000) // 10s
	defer cancel()

	store, err := redisstore.NewRedisStore(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis init: %v", err)
	}

	processor.Init(store)

	r := gin.Default()
	// Optional: lock down proxies for the warning in dev/prod
	// r.SetTrustedProxies([]string{"127.0.0.1"})
	r.POST("/events", api.HandleEvent)
	r.Run(":8080")
}
