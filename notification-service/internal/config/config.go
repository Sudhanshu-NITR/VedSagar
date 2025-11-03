package config

import (
	"os"
)

// Config holds minimal runtime settings; extend later as needed.
type Config struct {
	RedisURL string
	Port     string
}

// Load reads from environment variables (fallbacks provided)
func Load() Config {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		// local default (non-TLS)
		url = "redis://127.0.0.1:6379/0"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{
		RedisURL: url,
		Port:     port,
	}
}
