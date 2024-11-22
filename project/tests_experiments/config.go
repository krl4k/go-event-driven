package tests_experiments

import (
	"os"
	"time"
)

type TestConfig struct {
	UseLocalEnv bool
	ServiceURL  string
	RedisURL    string
	GatewayURL  string

	TestTimeout time.Duration
}

func LoadTestConfig() TestConfig {
	return TestConfig{
		UseLocalEnv: os.Getenv("USE_LOCAL_ENV") == "true",
		RedisURL:    os.Getenv("TEST_REDIS_URL"),
		ServiceURL:  os.Getenv("TEST_SERVICE_URL"),
		GatewayURL:  os.Getenv("TEST_GATEWAY_URL"),
		TestTimeout: getDurationEnv("TEST_TIMEOUT", 5*time.Minute),
	}
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
