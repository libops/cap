package config

import (
	"fmt"
	"os"
	"regexp"
	"time"
)

// Config holds all necessary configuration loaded from environment variables.
type Config struct {
	ProjectID      string
	Location       string
	Cluster        string
	CADVISORHost   string
	ScrapeInterval time.Duration
	FilterPattern  string
	FilterRegex    *regexp.Regexp
}

const (
	defaultCAdvisorHost   = "localhost:8080"
	defaultScrapeInterval = 30 * time.Second
	envVarName            = "SERVICE_PATTERN"
)

// LoadFromEnv reads configuration from environment variables.
func LoadFromEnv() (Config, error) {
	cfg := Config{
		ProjectID:      os.Getenv("GCP_PROJECT"),
		Location:       os.Getenv("GCP_ZONE"),
		Cluster:        os.Getenv("GCP_INSTANCE_NAME"),
		CADVISORHost:   os.Getenv("CADVISOR_HOST"),
		ScrapeInterval: defaultScrapeInterval,
	}

	if cfg.CADVISORHost == "" {
		cfg.CADVISORHost = defaultCAdvisorHost
	}

	cfg.FilterPattern = os.Getenv(envVarName)
	if cfg.FilterPattern == "" {
		cfg.FilterPattern = `.*`
	}

	var err error
	cfg.FilterRegex, err = regexp.Compile(cfg.FilterPattern)
	if err != nil {
		return cfg, fmt.Errorf("failed to compile regex pattern %q: %w", cfg.FilterPattern, err)
	}

	if cfg.ProjectID == "" || cfg.Location == "" || cfg.Cluster == "" {
		return cfg, fmt.Errorf("GCP environment variables (GCP_PROJECT, GCP_ZONE, GCP_INSTANCE_NAME) must be set")
	}

	return cfg, nil
}
