// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr     string
	GRPCAddr     string
	MongoURI     string
	MongoDB      string
	JWTSecret    string
	JWTIssuer    string
	JWTTTL       time.Duration
	CountEvery   time.Duration
	ShutdownWait time.Duration
}

// Load reads configuration from the environment, applying sensible defaults
// for everything except JWT_SECRET, which must be provided.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8080"),
		GRPCAddr:     getEnv("GRPC_ADDR", ":9090"),
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:      getEnv("MONGO_DB", "user_management"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		JWTIssuer:    getEnv("JWT_ISSUER", "user-management-api"),
		JWTTTL:       getEnvDuration("JWT_TTL", 24*time.Hour),
		CountEvery:   getEnvDuration("USER_COUNT_INTERVAL", 10*time.Second),
		ShutdownWait: getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
