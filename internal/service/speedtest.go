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
	updater := updater.New("0.0.0") // Start with 0.0.0 version, will be updated by GitHub API
	if updater == nil {
		return nil, fmt.Errorf("failed to create updater")
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
	url := fmt.Sprintf("%s/%s/main/presets/config.json", s.config.GithubRawBaseUrl, s.config.GithubRepo)

	// Get node data
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get node data: %w", err)
	}
	defer resp.Body.Close()

	// Read response data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response data: %w", err)
	}

	// Parse node data
	var tmpNodes models.NodeList
	if err := json.Unmarshal(data, &tmpNodes); err != nil {
		return fmt.Errorf("failed to parse node data: %w", err)
	}

	// Process node data
	s.nodes = make(models.NodeList, len(tmpNodes))
	for id, node := range tmpNodes {
		node.Size.Value = int64(node.Size.Value)
		s.nodes[id] = node
	}

	if len(s.nodes) == 0 {
		return fmt.Errorf("no available nodes")
	}

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
		"--type", node.Type,
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
