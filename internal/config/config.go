package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	MinIO MinIOConfig
	DB    DBConfig
	App   AppConfig
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type DBConfig struct {
	URL string
}

type AppConfig struct {
	Port      int
	Env       string
	JWTSecret string
}

func Load() *Config {
	return &Config{
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY",
			SecretKey: getEnv("MINIO_SECRET_KEY"),
			Bucket:    getEnv("MINIO_BUCKET"),
			UseSSL:    getEnvBool("MINIO_USE_SSL", false),
		},
		DB: DBConfig{
			URL: getEnv("DATABASE_URL", "postgres://appuser:apppassword@localhost:5432/minio_app?sslmode=disable"),
		},
		App: AppConfig{
			Port:      getEnvInt("APP_PORT", 8080),
			Env:       getEnv("APP_ENV", "development"),
			JWTSecret: getEnv("JWT_SECRET", "default-secret"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func (c *Config) String() string {
	return fmt.Sprintf("MinIO: %s, DB: %s, Port: %d", 
		c.MinIO.Endpoint, c.DB.URL, c.App.Port)
}