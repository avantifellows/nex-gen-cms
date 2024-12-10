package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Use a variable for log.Fatalf so that we can change its value while testing
var fatalf = log.Fatalf

// EnvLoader interface defines a method to load environment variables
type EnvLoader interface {
	Load() error
}

// this will implement EnvLoader
type Env struct{}

// Env implements EnvLoader here to load the environment variables using godotenv
func (p *Env) Load() error {
	return godotenv.Load()
}

// LoadEnv loads environment variables using the provided EnvLoader
func LoadEnv(loader EnvLoader) {
	err := loader.Load()
	if err != nil {
		fatalf("Error loading .env file")
	}
}

func GetEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}
