package service

import (
	"aqua-speed-tools/internal/config"
	"aqua-speed-tools/internal/models"
	"aqua-speed-tools/internal/updater"
	"aqua-speed-tools/internal/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// SpeedTest provides network speed testing functionality
type SpeedTest struct {
	config  config.Config    // Configuration information
	nodes   models.NodeList  // Node list
	updater *updater.Updater // Updater
}

// NewSpeedTest creates a new SpeedTest instance
func NewSpeedTest(cfg config.Config) (*SpeedTest, error) {
	updater, err := updater.NewWithLocalVersion("0.0.0") // Start with 0.0.0 version, will be updated by GitHub API
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}

	return &SpeedTest{
		config:  cfg,
		nodes:   make(models.NodeList),
		updater: updater,
	}, nil
}

// Init initializes the SpeedTest service
func (s *SpeedTest) Init() error {
	// Check for updates and get latest version
	if err := s.updater.CheckAndUpdate(); err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	// Initialize nodes
	return s.initNodes()
}

// initNodes initializes the speed test node list
func (s *SpeedTest) initNodes() error {
	url := fmt.Sprintf("%s/%s/main/presets/config.json", s.config.GithubRawBaseUrl, s.config.GithubToolsRepo)

	// Validate URL
	if url == "" {
		return fmt.Errorf("invalid empty URL")
	}

	// Get node data with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get node data from %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status code %d from %s", resp.StatusCode, url)
	}

	// Read response data with size limit
	const maxSize = 10 << 20 // 10 MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return fmt.Errorf("failed to read response data: %w", err)
	}

	// Validate JSON before parsing
	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON data received from %s", url)
	}

	// Parse node data
	var tmpNodes models.NodeList
	if err := json.Unmarshal(data, &tmpNodes); err != nil {
		// Log the actual JSON data for debugging
		truncatedData := string(data)
		if len(truncatedData) > 1000 {
			truncatedData = truncatedData[:1000] + "..."
		}
		return fmt.Errorf("failed to parse node data: %w\nReceived data: %s", err, truncatedData)
	}

	// Validate nodes before processing
	if err := tmpNodes.Validate(); err != nil {
		return fmt.Errorf("node validation failed: %w", err)
	}

	// Process node data with validation
	s.nodes = make(models.NodeList, len(tmpNodes))
	for id, node := range tmpNodes {
		// Validate ID
		if id == "" {
			return fmt.Errorf("empty node ID found")
		}

		// Validate node ID matches map key
		if node.Id != id {
			return fmt.Errorf("node ID mismatch: map key '%s' != node ID '%s'", id, node.Id)
		}

		// Validate size
		if node.Size.Value <= 0 {
			return fmt.Errorf("invalid size value for node %s: %d", id, node.Size.Value)
		}

		// Convert and store node
		node.Size.Value = int64(node.Size.Value)
		s.nodes[id] = node
	}

	// Validate final node list
	if len(s.nodes) == 0 {
		return fmt.Errorf("no valid nodes found in response")
	}

	// Log success
	utils.Green.Printf("Successfully loaded %d nodes\n", len(s.nodes))

	return nil
}

// ListNodes lists all available nodes
func (s *SpeedTest) ListNodes() error {
	if len(s.nodes) == 0 {
		return fmt.Errorf("node list is empty")
	}

	headers := []string{"Name", "ISP", "Node Type", "Required Traffic", "ID"}
	table := utils.NewTable(headers)

	table.EnableAutoMerge()
	table.SortBy([]string{"Node Type", "ISP"})

	for id, node := range s.nodes {
		// Calculate required traffic (4x consumption)
		size := humanize.Bytes(uint64(node.Size.Value) * 1000 * 1000 * 4)
		table.AddRow([]string{
			node.Name.Zh,
			node.Isp.Zh,
			strings.ToUpper(node.GeoInfo.Type),
			size,
			id,
		})
	}

	if len(s.nodes) > 25 {
		table.SetPageSize(25)
	}

	table.Print()
	return nil
}

// runSpeedTest runs speed test for a single node
func (s *SpeedTest) runSpeedTest(node models.Node) error {
	// Print test start information
	utils.Green.Printf("\n┌─────────────────────────────────────────┐\n")
	fmt.Printf("%s 🚀 Starting test for node: %s%s\n",
		utils.Green.Sprintf("│"),
		utils.Cyan.Sprint(node.Name.Zh),
		utils.Green.Sprintf(" "))
	utils.Green.Printf("└─────────────────────────────────────────┘\n\n")

	// Build test command
	cmdArgs := []string{
		"--thread", fmt.Sprintf("%d", node.Threads),
		"--server", node.Url,
		"--sn", node.Name.Zh,
		"--type", string(node.Type),
	}

	// Get binary path and execute command
	binaryPath := s.updater.GetBinaryPath()
	cmd := exec.Command(binaryPath, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("speed test failed: %w", err)
	}

	// Print test completion information
	utils.Green.Printf("\n┌─────────────────────────────────────────┐\n")
	fmt.Printf("%s 🎉 Test completed: %s%s\n",
		utils.Green.Sprintf("│"),
		utils.Cyan.Sprint(node.Name.Zh),
		utils.Green.Sprintf(" "))
	utils.Green.Printf("└─────────────────────────────────────────┘\n\n")

	return nil
}

// RunAllTest tests all nodes
func (s *SpeedTest) RunAllTest() error {
	if len(s.nodes) == 0 {
		return fmt.Errorf("no available nodes")
	}

	utils.Yellow.Println("Preparing to test all nodes...")
	for _, node := range s.nodes {
		if err := s.runSpeedTest(node); err != nil {
			return fmt.Errorf("failed to test node %s: %w", node.Name.Zh, err)
		}
	}
	utils.Green.Println(" ✨ All node tests completed")
	return nil
}

// RunTest tests a specified node
func (s *SpeedTest) RunTest(id string) error {
	node, ok := s.nodes[id]
	if !ok {
		utils.Red.Printf("Error: Invalid test ID: %s\n", id)
		utils.Yellow.Println("Use 'list' command to show all available nodes")
		fmt.Printf("%sAvailable test IDs: %s%v\n",
			utils.Blue.Sprint(""),
			utils.Cyan.Sprint(""),
			getAvailableIDs(s.nodes))
		return fmt.Errorf("invalid node ID: %s", id)
	}

	return s.runSpeedTest(node)
}

// getAvailableIDs gets all available node IDs
func getAvailableIDs(nodes models.NodeList) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	return ids
}
