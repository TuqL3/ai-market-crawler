package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL      string
	PythonGRPCAddr   string
	APIPort          string
	GitHubToken      string
	GoCrawlerEnabled bool
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	pythonAddr := os.Getenv("PYTHON_GRPC_ADDR")
	if pythonAddr == "" {
		pythonAddr = "localhost:50052"
	}

	apiPort := os.Getenv("GO_API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	gitHubToken := os.Getenv("GITHUB_TOKEN")
	if gitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required")
	}

	goCrawlerEnabled := os.Getenv("GO_CRAWLER_ENABLED")
	if goCrawlerEnabled == "" {
		goCrawlerEnabled = "true"
	}

	return &Config{
		DatabaseURL:      dbURL,
		PythonGRPCAddr:   pythonAddr,
		APIPort:          apiPort,
		GitHubToken:      gitHubToken,
		GoCrawlerEnabled: goCrawlerEnabled != "false",
	}, nil
}
