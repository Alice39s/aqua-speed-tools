package utils

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHubURLs contains all GitHub related URLs
type GitHubURLs struct {
	RawBaseURL    string
	APIURL        string
	FastestMirror string // The fastest mirror URL for Release downloads
}

// NewGitHubURLs creates a new GitHubURLs instance
func NewGitHubURLs(rawMagicURL, apiMagicURL string, rawJsdelivrSet []string) *GitHubURLs {
	urls := &GitHubURLs{
		RawBaseURL: "https://raw.githubusercontent.com",
		APIURL:     "https://api.github.com",
	}

	// If Raw Magic URL is provided, use it
	if rawMagicURL != "" {
		urls.RawBaseURL = normalizeURL(rawMagicURL)
		urls.FastestMirror = normalizeURL(rawMagicURL)
	} else if len(rawJsdelivrSet) > 0 {
		// Otherwise, try to find the best available URL from the set
		if bestURL := findBestRawURL(rawJsdelivrSet); bestURL != "" {
			urls.RawBaseURL = normalizeURL(bestURL)
			urls.FastestMirror = normalizeURL(bestURL)
		}
	}

	// If API Magic URL is provided, use it
	if apiMagicURL != "" {
		urls.APIURL = normalizeURL(apiMagicURL)
	}

	return urls
}

// normalizeURL ensures the URL doesn't end with a slash
func normalizeURL(u string) string {
	return strings.TrimRight(u, "/")
}

// isURLAccessible checks if a URL is accessible
func isURLAccessible(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		LogWarning("Invalid URL: %v", err)
		return false
	}

	// If DNS resolver is set, use it
	if resolver := GetDNSResolver(); resolver != nil {
		ips, err := resolver.Resolve(parsedURL.Hostname())
		if err != nil {
			LogWarning("DNS resolution failed for %s: %v", parsedURL.Hostname(), err)
			return false
		}
		// Consider accessible if we can resolve the IP
		return len(ips) > 0
	}

	// Otherwise use system default DNS resolver
	ips, err := net.LookupIP(parsedURL.Hostname())
	if err != nil {
		LogWarning("DNS lookup failed for %s: %v", parsedURL.Hostname(), err)
		return false
	}

	return len(ips) > 0
}

// BuildRawURL builds a raw content URL for GitHub
func (u *GitHubURLs) BuildRawURL(owner, repo, branch, path string) string {
	parts := []string{u.RawBaseURL}
	if owner != "" {
		parts = append(parts, owner)
	}
	if repo != "" {
		parts = append(parts, repo)
	}
	if branch != "" {
		parts = append(parts, branch)
	}
	if path != "" {
		parts = append(parts, path)
	}
	return strings.Join(parts, "/")
}

// BuildAPIURL builds an API URL for GitHub
func (u *GitHubURLs) BuildAPIURL(path string) string {
	// 如果是自定义 API URL，直接拼接路径
	if u.APIURL != "https://api.github.com" {
		return fmt.Sprintf("%s/%s", u.APIURL, strings.TrimPrefix(path, "/"))
	}
	// 官方 API 需要 /repos/ 前缀
	return fmt.Sprintf("%s/repos/%s", u.APIURL, strings.TrimPrefix(path, "/"))
}

// ConvertReleaseURLToMirror converts a GitHub release URL to a mirror URL
func ConvertReleaseURLToMirror(releaseURL, mirrorBaseURL string) (string, error) {
	// Parse the release URL: https://github.com/owner/repo/releases/download/tag/filename
	parsedURL, err := url.Parse(releaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid release URL: %w", err)
	}

	// Check if it's a GitHub release URL
	if parsedURL.Host != "github.com" {
		return releaseURL, nil // Return original if not from GitHub
	}

	// Extract parts from path: /owner/repo/releases/download/tag/filename
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 6 || pathParts[2] != "releases" || pathParts[3] != "download" {
		return releaseURL, nil // Return original if not a release download URL
	}

	owner := pathParts[0]
	repo := pathParts[1]
	tag := pathParts[4]
	filename := strings.Join(pathParts[5:], "/")

	// Convert to jsDelivr format if mirror is jsDelivr
	if strings.Contains(mirrorBaseURL, "jsdelivr.net") {
		return fmt.Sprintf("%s/%s/%s@%s/%s", mirrorBaseURL, owner, repo, tag, filename), nil
	}

	// For other mirrors, we'll assume they don't support release files directly
	return releaseURL, nil
}

// findBestRawURL tests and returns the fastest available Raw URL
func findBestRawURL(urls []string) string {
	type result struct {
		url     string
		latency time.Duration
	}

	results := make(chan result, len(urls))

	// Test all URLs
	for _, url := range urls {
		go func(u string) {
			start := time.Now()
			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			var bestLatency time.Duration
			var success bool

			for attempt := 0; attempt <= 3; attempt++ {
				if attempt > 0 {
					LogDebug("Retrying Raw URL %s: attempt %d/3", u, attempt)
					time.Sleep(time.Second * time.Duration(attempt))
				}

				resp, err := client.Get(u)
				if err != nil {
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					latency := time.Since(start)
					if bestLatency == 0 || latency < bestLatency {
						bestLatency = latency
					}
					success = true
					break
				}
			}

			if success {
				results <- result{url: u, latency: bestLatency}
			} else {
				results <- result{url: u, latency: time.Hour} // Use large latency for failed URLs
			}
		}(url)
	}

	// Collect results
	var bestURL string
	var bestLatency time.Duration

	for i := 0; i < len(urls); i++ {
		r := <-results
		if r.latency >= time.Hour {
			LogWarning("Failed to test Raw URL: %s", r.url)
			continue
		}

		LogDebug("Raw URL %s latency: %v", r.url, r.latency)

		if bestURL == "" || r.latency < bestLatency {
			bestURL = r.url
			bestLatency = r.latency
		}
	}

	if bestURL != "" {
		LogInfo("Selected Raw URL: %s (latency: %v)", bestURL, bestLatency)
	}

	return bestURL
}
