package service

import (
	"aqua-speed-tools/internal/models"
	"aqua-speed-tools/internal/updater"
	"aqua-speed-tools/internal/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"go.uber.org/zap"
)

type TestService struct {
	nodes   []models.Node
	logger  *zap.Logger
	updater *updater.Updater
}

func NewTestService(nodes []models.Node, logger *zap.Logger, updater *updater.Updater) *TestService {
	return &TestService{
		nodes:   nodes,
		logger:  logger,
		updater: updater,
	}
}

func (s *TestService) RunAllTest() error {
	if len(s.nodes) == 0 {
		s.logger.Error("no available nodes")
		return fmt.Errorf("no available nodes")
	}

	s.logger.Info("starting test for all nodes")
	utils.Yellow.Println("Preparing to test all nodes...")

	for _, node := range s.nodes {
		if err := s.runSpeedTest(node); err != nil {
			s.logger.Error("failed to test node",
				zap.String("node", node.Name.Zh),
				zap.Error(err))
			return fmt.Errorf("failed to test node %s: %w", node.Name.Zh, err)
		}
	}

	s.logger.Info("all node tests completed successfully")
	utils.Green.Println(" ✨ All node tests completed")
	return nil
}

func (s *TestService) RunTest(input string) error {
	var numID int
	if _, err := fmt.Sscanf(input, "%d", &numID); err == nil {
		// Try to find the node by numeric ID
		index := 1
		// Sort nodes by type and ISP to match table display
		sortedNodes := getSortedNodes(s.nodes)
		for _, node := range sortedNodes {
			if index == numID {
				return s.runSpeedTest(node)
			}
			index++
		}
		s.logger.Error("invalid numeric ID provided",
			zap.Int("id", numID))
		utils.Red.Printf("Error: Invalid numeric ID: %d\n", numID)
		utils.Yellow.Println("Use 'list' command to show all available nodes")
		return fmt.Errorf("invalid numeric ID: %d", numID)
	}

	// If not a number, treat as a node ID
	node, ok := s.getNodeByID(input)
	if !ok {
		s.logger.Error("invalid node ID provided",
			zap.String("id", input))
		utils.Red.Printf("Error: Invalid test ID: %s\n", input)
		utils.Yellow.Println("Use 'list' command to show all available nodes")
		fmt.Printf("%sAvailable test IDs: %s%v\n",
			utils.Blue.Sprint(""),
			utils.Cyan.Sprint(""),
			getAvailableIDs(s.nodes))
		return fmt.Errorf("invalid node ID: %s", input)
	}

	return s.runSpeedTest(node)
}

func (s *TestService) runSpeedTest(node models.Node) error {
	s.logger.Info("starting speed test for node",
		zap.String("node", node.Name.Zh))

	printTestHeader(node)

	if err := s.executeTest(node); err != nil {
		s.logger.Error("speed test execution failed",
			zap.String("node", node.Name.Zh),
			zap.Error(err))
		return err
	}

	// s.logger.Info("speed test completed successfully",
	// 	zap.String("node", node.Name.Zh))
	printTestFooter(node)
	return nil
}

func (s *TestService) executeTest(node models.Node) error {
	cmdArgs := []string{
		"--thread", fmt.Sprintf("%d", node.Threads),
		"--server", node.Url,
		"--sn", node.Name.Zh,
		"--type", string(node.Type),
	}

	binaryPath := filepath.Join(s.updater.InstallDir, "bin", s.updater.BinaryName)
	cmd := exec.Command(binaryPath, cmdArgs...)

	s.logger.Info("executing speed test command",
		zap.String("binary", binaryPath),
		zap.String("node", node.Name.Zh),
		zap.Strings("args", cmdArgs))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		s.logger.Error("command execution failed",
			zap.String("binary", binaryPath),
			zap.String("node", node.Name.Zh),
			zap.Error(err))
	}
	return err
}

func (s *TestService) getNodeByID(id string) (models.Node, bool) {
	for _, node := range s.nodes {
		if node.Id == id {
			return node, true
		}
	}
	return models.Node{}, false
}

func printTestHeader(node models.Node) {
	utils.Green.Printf("\n┌─────────────────────────────────────────┐\n")
	fmt.Printf("%s 🚀 Starting test for node: %s%s\n",
		utils.Green.Sprintf("│"),
		utils.Cyan.Sprint(node.Name.Zh),
		utils.Green.Sprintf(" "))
	utils.Green.Printf("└─────────────────────────────────────────┘\n\n")
}

func printTestFooter(node models.Node) {
	utils.Green.Printf("\n┌─────────────────────────────────────────┐\n")
	fmt.Printf("%s 🎉 Test completed: %s%s\n",
		utils.Green.Sprintf("│"),
		utils.Cyan.Sprint(node.Name.Zh),
		utils.Green.Sprintf(" "))
	utils.Green.Printf("└─────────────────────────────────────────┘\n\n")
}

// getSortedNodes returns nodes sorted by type and ISP to match table display
func getSortedNodes(nodes []models.Node) []models.Node {
	sortedNodes := make([]models.Node, len(nodes))
	copy(sortedNodes, nodes)

	// Sort by type and ISP to match table display
	sort.Slice(sortedNodes, func(i, j int) bool {
		if sortedNodes[i].GeoInfo.Type != sortedNodes[j].GeoInfo.Type {
			return sortedNodes[i].GeoInfo.Type < sortedNodes[j].GeoInfo.Type
		}
		return sortedNodes[i].Isp.Zh < sortedNodes[j].Isp.Zh
	})

	return sortedNodes
}
