package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
// We load once at startup and pass this struct around — no global state,
// no os.Getenv() scattered throughout the codebase.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	AWS      AWSConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// DSN builds the MySQL Data Source Name for Ent.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC",
		d.User, d.Password, d.Host, d.Port, d.Name,
	)
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

type AWSConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	S3Bucket        string
}

// Load reads .env and returns a Config.
// Call this once in main.go.
func Load() (*Config, error) {
	// In production (ENV=production), env vars come from the host.
	// .env is only for local dev. godotenv.Load doesn't overwrite existing env vars.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	expiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     mustGetEnv("DB_USER"),
			Password: mustGetEnv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME", "cerberus"),
		},
		JWT: JWTConfig{
			Secret:      mustGetEnv("JWT_SECRET"),
			ExpiryHours: expiryHours,
		},
		AWS: AWSConfig{
			AccessKeyID:     mustGetEnv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: mustGetEnv("AWS_SECRET_ACCESS_KEY"),
			Region:          getEnv("AWS_REGION", "ap-south-1"),
			S3Bucket:        mustGetEnv("AWS_S3_BUCKET"),
		},
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// mustGetEnv panics at startup if a required variable is missing.
// Fail fast at boot is better than mysterious nil pointer panics in production.
func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("FATAL: required environment variable %q is not set", key)
	}
	return val
}
