package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aqua-speed-tools/internal/github"
)

// Config represents the application configuration
type Config struct {
	Script            ScriptConfig         `json:"script"`
	GitHubRawMagicSet []string             `json:"github_raw_magic_set"`
	DNSOverHTTPSSet   []DNSOverHTTPSConfig `json:"dns_over_https_set"`
	GithubRawBaseUrl  string               `json:"githubRawBaseUrl"`
	GithubApiBaseUrl  string               `json:"githubApiBaseUrl"`
	GithubRepo        string               `json:"githubRepo"`
	GithubToolsRepo   string               `json:"githubToolsRepo"`
	DownloadTimeout   int                  `json:"downloadTimeout"`
}

// ScriptConfig represents the script configuration
type ScriptConfig struct {
	Version string `json:"version"`
	Prefix  string `json:"prefix"`
}

// DNSOverHTTPSConfig represents the DNS over HTTPS configuration
type DNSOverHTTPSConfig struct {
	Endpoint string `json:"endpoint"`
	Timeout  int    `json:"timeout"`
	Retries  int    `json:"retries"`
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("Configuration error: %s - %s", e.Field, e.Message)
}

var (
	// ConfigReader is the global configuration reader
	ConfigReader = &Config{}
)

// LoadConfig loads the configuration from a file
func LoadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果配置文件不存在，尝试从远程获取默认配置
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			client := github.NewClient(nil, "", "")
			data, err = client.GetDefaultConfig(ctx, "alice39s", "aqua-speed-tools")
			if err != nil {
				return fmt.Errorf("failed to download default config: %w", err)
			}

			if err := os.WriteFile(configPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write default config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	if err := json.Unmarshal(data, ConfigReader); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(ConfigReader); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Validate Script
	if cfg.Script.Version == "" {
		return &ConfigError{Field: "Script.Version", Message: "cannot be empty"}
	}
	if cfg.Script.Prefix == "" {
		return &ConfigError{Field: "Script.Prefix", Message: "cannot be empty"}
	}

	// Validate GitHub configuration
	if cfg.GithubRepo == "" {
		return &ConfigError{Field: "GithubRepo", Message: "cannot be empty"}
	}
	if cfg.GithubToolsRepo == "" {
		return &ConfigError{Field: "GithubToolsRepo", Message: "cannot be empty"}
	}

	// Validate GitHubRawMagicSet
	if len(cfg.GitHubRawMagicSet) == 0 {
		return &ConfigError{Field: "GitHubRawMagicSet", Message: "must contain at least one URL"}
	}
	for i, magic := range cfg.GitHubRawMagicSet {
		if magic == "" {
			return &ConfigError{Field: fmt.Sprintf("GitHubRawMagicSet[%d]", i), Message: "cannot be empty"}
		}
	}

	// Validate DNSOverHTTPSSet
	for i, doh := range cfg.DNSOverHTTPSSet {
		if doh.Endpoint == "" {
			return &ConfigError{Field: fmt.Sprintf("DNSOverHTTPSSet[%d].Endpoint", i), Message: "cannot be empty"}
		}
		if doh.Timeout <= 0 {
			return &ConfigError{Field: fmt.Sprintf("DNSOverHTTPSSet[%d].Timeout", i), Message: "must be greater than 0"}
		}
		if doh.Retries < 0 {
			return &ConfigError{Field: fmt.Sprintf("DNSOverHTTPSSet[%d].Retries", i), Message: "cannot be negative"}
		}
	}

	// Validate DownloadTimeout
	if cfg.DownloadTimeout <= 0 {
		return &ConfigError{Field: "DownloadTimeout", Message: "must be greater than 0"}
	}

	return nil
}
