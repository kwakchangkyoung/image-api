package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	UploadDir   string
	MaxFileSize int64
	AllowedMIME map[string]string
	CORSOrigins []string
	BaseURL     string
	Port        string
	APIKey      string
}

func Load() *Config {
	_ = godotenv.Load()

	uploadDir := getEnv("UPLOAD_DIR", "/data/images")
	maxFileSize := getEnvAsInt64("MAX_FILE_SIZE", 20971520)
	baseURL := getEnv("BASE_URL", "http://localhost:8080")
	port := getEnv("PORT", "8080")
	apiKey := os.Getenv("API_KEY")
	corsOrigins := parseOrigins(getEnv("CORS_ORIGINS", "*"))

	return &Config{
		UploadDir:   uploadDir,
		MaxFileSize: maxFileSize,
		AllowedMIME: map[string]string{
			"image/jpeg": "jpg",
			"image/png":  "png",
			"image/gif":  "gif",
			"image/webp": "webp",
		},
		CORSOrigins: corsOrigins,
		BaseURL:     baseURL,
		Port:        port,
		APIKey:      apiKey,
	}
}

func getEnv(key string, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		origins = append(origins, trimmed)
	}

	if len(origins) == 0 {
		return []string{"*"}
	}

	return origins
}
