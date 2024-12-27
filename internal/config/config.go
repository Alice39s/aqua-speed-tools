package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type BinaryConfig struct {
	Prefix string `json:"prefix"`
}

type ScriptConfig struct {
	Version string `json:"version"`
	Prefix  string `json:"prefix"`
}

type Config struct {
	Binary           BinaryConfig `json:"binary"`
	Script           ScriptConfig `json:"script"`
	GithubBaseUrl    string       `json:"githubBaseUrl"`
	GithubApiBaseUrl string       `json:"githubApiBaseUrl"`
	GithubRawBaseUrl string       `json:"githubRawBaseUrl"`
	GithubRepo       string       `json:"githubRepo"`
	GithubToolsRepo  string       `json:"githubToolsRepo"`
	TablePadding     int          `json:"tablePadding"`
	LogLevel         string       `json:"logLevel"`
}

// Configuration error type
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("Configuration error: %s - %s", e.Field, e.Message)
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile("configs/base.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg.Binary.Prefix == "" {
		return &ConfigError{Field: "Binary.Prefix", Message: "cannot be empty"}
	}
	if cfg.Script.Version == "" {
		return &ConfigError{Field: "Script.Version", Message: "cannot be empty"}
	}
	if cfg.Script.Prefix == "" {
		return &ConfigError{Field: "Script.Prefix", Message: "cannot be empty"}
	}
	if cfg.GithubBaseUrl == "" {
		return &ConfigError{Field: "GithubBaseUrl", Message: "cannot be empty"}
	}
	if cfg.GithubApiBaseUrl == "" {
		return &ConfigError{Field: "GithubApiBaseUrl", Message: "cannot be empty"}
	}
	if cfg.GithubRawBaseUrl == "" {
		return &ConfigError{Field: "GithubRawBaseUrl", Message: "cannot be empty"}
	}
	if cfg.GithubRepo == "" {
		return &ConfigError{Field: "GithubRepo", Message: "cannot be empty"}
	}
	if cfg.GithubToolsRepo == "" {
		return &ConfigError{Field: "GithubToolsRepo", Message: "cannot be empty"}
	}
	if cfg.TablePadding < 0 || cfg.TablePadding > 10 {
		return &ConfigError{Field: "TablePadding", Message: "cannot be negative or greater than 10"}
	}
	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.LogLevel] {
		return &ConfigError{Field: "LogLevel", Message: "must be debug, info, warn, or error"}
	}
	return nil
}

var ConfigReader = func() Config {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := validateConfig(cfg); err != nil {
		fmt.Printf("Config validation failed: %v\n", err)
		os.Exit(1)
	}

	return *cfg
}()
