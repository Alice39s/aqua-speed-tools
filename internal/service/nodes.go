package service

import (
	"aqua-speed-tools/internal/config"
	"aqua-speed-tools/internal/models"
	"aqua-speed-tools/internal/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// initNodes initializes the speed test node list
func (s *SpeedTest) initNodes() error {
	owner, repo := splitRepo(config.DefaultGithubToolsRepo)

	var url string
	if len(s.config.GithubRawJsdelivrSet) > 0 {
		mirrorURL := s.config.GithubRawJsdelivrSet[0]
		url = fmt.Sprintf("%s/%s/%s@main/presets/config.json",
			strings.TrimSuffix(mirrorURL, "/"),
			owner,
			repo)
	} else {
		url = fmt.Sprintf("%s/%s/%s/main/presets/config.json",
			s.config.GithubRawBaseURL,
			owner,
			repo)
	}

	// Validate URL
	if url == "" {
		return fmt.Errorf("invalid empty URL")
	}

	nodeData, err := s.fetchNodeData(url)
	if err != nil {
		return err
	}

	if err := s.parseAndValidateNodes(nodeData); err != nil {
		return err
	}

	// Log success
	utils.Green.Printf("Successfully loaded %d nodes\n", len(s.nodes))

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

func (s *SpeedTest) fetchNodeData(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set proper User-Agent header
	req.Header.Set("User-Agent", utils.GetUserAgent("Aqua-Speed-Tools"))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get node data from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status code %d from %s", resp.StatusCode, url)
	}

	const maxSize = 10 << 20 // 10 MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response data: %w", err)
	}

	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON data received from %s", url)
	}

	return data, nil
}

func (s *SpeedTest) parseAndValidateNodes(data []byte) error {
	var tmpNodes models.NodeList
	if err := json.Unmarshal(data, &tmpNodes); err != nil {
		truncatedData := string(data)
		if len(truncatedData) > 1000 {
			truncatedData = truncatedData[:1000] + "..."
		}
		return fmt.Errorf("failed to parse node data: %w\nReceived data: %s", err, truncatedData)
	}

	if err := tmpNodes.Validate(); err != nil {
		return fmt.Errorf("node validation failed: %w", err)
	}

	return s.processNodes(tmpNodes)
}

func (s *SpeedTest) processNodes(tmpNodes models.NodeList) error {
	s.nodes = make(models.NodeList, len(tmpNodes))
	for id, node := range tmpNodes {
		if err := validateNode(id, node); err != nil {
			return err
		}

		node.Size.Value = int64(node.Size.Value)
		s.nodes[id] = node
	}

	if len(s.nodes) == 0 {
		return fmt.Errorf("no valid nodes found in response")
	}

	return nil
}

func validateNode(id string, node models.Node) error {
	if id == "" {
		return fmt.Errorf("empty node ID found")
	}

	if node.Id != id {
		return fmt.Errorf("node ID mismatch: map key '%s' != node ID '%s'", id, node.Id)
	}

	if node.Size.Value <= 0 {
		return fmt.Errorf("invalid size value for node %s: %d", id, node.Size.Value)
	}

	return nil
}
