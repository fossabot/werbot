package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Load reads .env file configurations into ENV
func Load(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if err := godotenv.Load(path); err != nil {
		return err
	}

	return nil
}

func lookup(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// GetString is ...
func GetString(key, fallback string) string {
	return lookup(key, fallback)
}

// GetSliceString is ...
// TODO: дописать
func GetSliceString(key, fallback string) []string {
	var slice []string
	return slice
}

// GetInt is ...
func GetInt(key string, fallback int) int {
	value := lookup(key, "")
	if value, err := strconv.Atoi(value); err == nil {
		return value
	}
	return fallback
}

// GetBool is ...
func GetBool(key string, fallback bool) bool {
	value := lookup(key, "")
	if value, err := strconv.ParseBool(value); err == nil {
		return value
	}
	return fallback
}

// GetDuration is ...
func GetDuration(key, fallback string) time.Duration {
	value := lookup(key, fallback)
	duration, _ := time.ParseDuration(value)
	return duration
}

// GetByteFromFile is ...
func GetByteFromFile(key, fallback string) []byte {
	var data []byte
	value := lookup(key, fallback)

	if data = readFile(value); data != nil {
		return data
	}

	return nil
}

func readFile(file string) []byte {
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Print(err)
		return nil
	}
	return data
}
