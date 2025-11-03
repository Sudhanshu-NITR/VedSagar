package config

import "os"

type Config struct {
	RedisURL string
}

func Load() Config {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		// Local dev fallback (no TLS) if you run Docker redis:latest
		url = "redis://127.0.0.1:6379"
	}
	return Config{RedisURL: url}
}
