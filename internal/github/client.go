package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client represents a GitHub API client
type Client struct {
	httpClient *http.Client
	baseURL    string
	rawBaseURL string
}

// NewClient creates a new GitHub client
func NewClient(httpClient *http.Client, baseURL, rawBaseURL string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	if rawBaseURL == "" {
		rawBaseURL = "https://raw.githubusercontent.com"
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		rawBaseURL: rawBaseURL,
	}
}

// GetDefaultConfig fetches the default configuration from GitHub
func (c *Client) GetDefaultConfig(ctx context.Context, owner, repo string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s/main/configs/base.json", c.rawBaseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch config: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// GetLatestRelease fetches the latest release information
func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest release: HTTP %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return release.TagName, nil
}

// GetRawContent fetches raw content from GitHub
func (c *Client) GetRawContent(ctx context.Context, owner, repo, branch, filepath string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", c.rawBaseURL, owner, repo, branch, filepath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch content: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}
