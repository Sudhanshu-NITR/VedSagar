package main

import (
	"context"
	"log"
	"notification-service/internal/api"
	redisstore "notification-service/internal/storage/redis"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()
	store, err := redisstore.NewRedisStore(ctx, redisstore.Config{
		Addr:     os.Getenv("REDIS_ADDR"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		UseTLS:   false, // set true if using rediss://
	})
	if err != nil {
		log.Fatalf("redis init: %v", err)
	}
	defer store.Close(ctx)

	r := gin.Default()
	r.POST("/events", api.HandleEvent)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
