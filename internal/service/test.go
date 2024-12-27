package service

import (
	"aqua-speed-tools/internal/models"
	"aqua-speed-tools/internal/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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

// runSpeedTest runs speed test for a single node
func (s *SpeedTest) runSpeedTest(node models.Node) error {
	printTestHeader(node)

	if err := s.executeTest(node); err != nil {
		return err
	}

	printTestFooter(node)
	return nil
}

func (s *SpeedTest) executeTest(node models.Node) error {
	cmdArgs := []string{
		"--thread", fmt.Sprintf("%d", node.Threads),
		"--server", node.Url,
		"--sn", node.Name.Zh,
		"--type", string(node.Type),
	}

	binaryPath := filepath.Join(s.updater.InstallDir, "bin", s.updater.BinaryName)
	cmd := exec.Command(binaryPath, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
