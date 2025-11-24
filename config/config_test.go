package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/libops/cap/config"
)

// Helper function to reset environment variables after each test
func resetEnv() {
	_ = os.Unsetenv("GCP_PROJECT")
	_ = os.Unsetenv("GCP_ZONE")
	_ = os.Unsetenv("GCP_INSTANCE_NAME")
	_ = os.Unsetenv("CADVISOR_HOST")
	_ = os.Unsetenv("SERVICE_PATTERN")
}

func TestLoadFromEnv_Success(t *testing.T) {
	resetEnv()
	_ = os.Setenv("GCP_PROJECT", "test-project")
	_ = os.Setenv("GCP_ZONE", "us-central1-a")
	_ = os.Setenv("GCP_INSTANCE_NAME", "test-cluster")
	_ = os.Setenv("SERVICE_PATTERN", `(test-service|other-service)`)

	cfg, err := config.LoadFromEnv()

	if err != nil {
		t.Fatalf("LoadFromEnv failed unexpectedly: %v", err)
	}

	if cfg.ProjectID != "test-project" {
		t.Errorf("Expected ProjectID 'test-project', got %s", cfg.ProjectID)
	}
	if cfg.CADVISORHost != "localhost:8080" {
		t.Errorf("Expected default CADVISORHost 'localhost:8080', got %s", cfg.CADVISORHost)
	}
	if cfg.FilterRegex.String() != `(test-service|other-service)` {
		t.Errorf("Expected regex to match pattern, got %s", cfg.FilterRegex.String())
	}
}

func TestLoadFromEnv_MissingGCPVars(t *testing.T) {
	resetEnv()
	// Intentionally omit GCP_PROJECT

	_, err := config.LoadFromEnv()

	if err == nil {
		t.Fatal("LoadFromEnv unexpectedly succeeded when required GCP vars were missing")
	}
	expectedError := "GCP environment variables (GCP_PROJECT, GCP_ZONE, GCP_INSTANCE_NAME) must be set"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestLoadFromEnv_InvalidRegex(t *testing.T) {
	resetEnv()
	_ = os.Setenv("GCP_PROJECT", "p")
	_ = os.Setenv("GCP_ZONE", "z")
	_ = os.Setenv("GCP_INSTANCE_NAME", "c")
	// Invalid regex: trailing backslash
	_ = os.Setenv("SERVICE_PATTERN", `\`)

	_, err := config.LoadFromEnv()

	if err == nil {
		t.Fatal("LoadFromEnv unexpectedly succeeded with invalid regex")
	}
	expectedError := "failed to compile regex pattern"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}
