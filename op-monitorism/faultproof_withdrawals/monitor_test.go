//go:build live
// +build live

package faultproof_withdrawals

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// loadEnv loads environment variables from the specified .env file.
func loadEnv(env string) (envMap map[string]string, err error) {

	// Read the file content
	content, err := os.ReadFile(env)
	if err != nil {
		log.Fatalf("Error reading .env file: %v", err)
	}

	// Unmarshal the content (pass it as string)
	return godotenv.Unmarshal(string(content))
}
