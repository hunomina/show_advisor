package config

import (
	"os"
)

type Config struct {
	QdrantURL      string
	OllamaURL      string
	HttpApiPort    string
	Model          string
	CollectionName string
}

func Load() *Config {
	return &Config{
		QdrantURL:      getEnv("QDRANT_URL", "http://localhost:6333"),
		OllamaURL:      getEnv("OLLAMA_URL", "http://localhost:11434"),
		HttpApiPort:    getEnv("HTTP_API_PORT", "8080"),
		Model:          getEnv("MODEL", "nomic-embed-text"),
		CollectionName: getEnv("COLLECTION", "shows"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
