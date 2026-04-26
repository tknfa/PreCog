package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func loadDotEnvIfPresent() {
	file, err := os.Open(".env")
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Warning: failed to open .env file: %v", err)
		}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" || value == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			log.Printf("Warning: failed to set env var %s from .env: %v", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Warning: failed to read .env file: %v", err)
	}
}
