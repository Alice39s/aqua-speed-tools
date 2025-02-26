package updater

import (
	"aqua-speed-tools/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// GitHubRelease represents the GitHub release API response.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// GitHubClient defines the interface for GitHub operations.
type GitHubClient interface {
	GetLatestRelease(ctx context.Context, apiURL string) (*GitHubRelease, error)
	GetRawContent(ctx context.Context, rawURL string) ([]byte, error)
}

// DefaultGitHubClient is the default implementation of GitHubClient.
type DefaultGitHubClient struct {
	client  *http.Client
	logger  *zap.Logger
	version string
	urls    *utils.GitHubURLs
}

// NewDefaultGitHubClient creates a new DefaultGitHubClient instance.
func NewDefaultGitHubClient(client *http.Client, logger *zap.Logger, version string, urls *utils.GitHubURLs) *DefaultGitHubClient {
	return &DefaultGitHubClient{
		client:  client,
		logger:  logger,
		version: version,
		urls:    urls,
	}
}

// GetLatestRelease fetches the latest release from the GitHub API.
func (c *DefaultGitHubClient) GetLatestRelease(ctx context.Context, apiURL string) (*GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := "Aqua-Speed-Updater/" + c.version
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		resetTime := resp.Header.Get("X-RateLimit-Reset")
		return nil, fmt.Errorf("rate limit exceeded, reset at: %s", resetTime)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return &release, nil
}

// GetRawContent fetches raw content from GitHub.
func (c *DefaultGitHubClient) GetRawContent(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := "Aqua-Speed-Updater/" + c.version
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub returned status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 限制为 10MB
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// GetDefaultConfig fetches the default configuration from GitHub.
func (c *DefaultGitHubClient) GetDefaultConfig(ctx context.Context, owner, repo string) ([]byte, error) {
	rawURL := c.urls.BuildRawURL(owner, repo, "main", "configs/base.json")
	c.logger.Debug("Fetching default config", zap.String("url", rawURL))

	return c.GetRawContent(ctx, rawURL)
}
