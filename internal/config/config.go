package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"aqua-speed-tools/internal/github"
)

// Config represents the application configuration
type Config struct {
	Script               ScriptConfig         `json:"script"`
	GithubRawJsdelivrSet []string             `json:"github_raw_jsdelivr_set"`
	DNSOverHTTPSSet      []DNSOverHTTPSConfig `json:"dns_over_https_set"`
	GithubRawBaseURL     string               `json:"github_raw_base_url"`
	GithubAPIBaseURL     string               `json:"github_api_base_url"`
	GithubAPIMagicURL    string               `json:"github_api_magic_url"`
	TablePadding         int                  `json:"table_padding"`
	LogLevel             string               `json:"log_level"`
	DownloadTimeout      int                  `json:"download_timeout"`
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

	// 硬编码的仓库信息
	DefaultGithubRepo      = "alice39s/aqua-speed"
	DefaultGithubToolsRepo = "alice39s/aqua-speed-tools"
)

// GetConfigDir returns the configuration directory based on the operating system
func GetConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "aqua-speed-tools")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "aqua-speed-tools")
	default: // linux and others
		if os.Getuid() == 0 {
			return "/etc/aqua-speed-tools"
		}
		return filepath.Join(os.Getenv("HOME"), ".config", "aqua-speed-tools")
	}
}

// LoadConfig loads the configuration from a file
func LoadConfig(configPath string) error {
	// 如果没有指定配置路径，使用默认路径
	if configPath == "" {
		configPath = filepath.Join(GetConfigDir(), "base.json")
	}

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
			owner, repo := splitRepo(DefaultGithubToolsRepo)
			data, err = client.GetDefaultConfig(ctx, owner, repo)
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

	// Validate GitHubRawJsdelivrSet
	if len(cfg.GithubRawJsdelivrSet) == 0 {
		return &ConfigError{Field: "GitHubRawJsdelivrSet", Message: "must contain at least one URL"}
	}
	for i, jsdelivr := range cfg.GithubRawJsdelivrSet {
		if jsdelivr == "" {
			return &ConfigError{Field: fmt.Sprintf("GitHubRawJsdelivrSet[%d]", i), Message: "cannot be empty"}
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

// splitRepo splits a repository string into owner and repo parts
func splitRepo(fullRepo string) (owner, repo string) {
	parts := strings.Split(fullRepo, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
