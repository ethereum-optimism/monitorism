package faultproof_withdrawals

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// TestMain runs the tests in the package and exits with the appropriate exit code.
func TestMain(m *testing.M) {
	exitVal := m.Run()
	os.Exit(exitVal)
}

// loadEnv loads environment variables from the specified .env file.
func loadEnv(env string) error {
	return godotenv.Load(env)
}
