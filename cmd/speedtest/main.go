package main

import (
	"aqua-speed-tools/internal/config"
	"aqua-speed-tools/internal/service"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

const (
	// Program description
	programDesc = "Network Speed Test Tool - Supports testing network speed for specific nodes or all nodes"
	// Command usage instructions
	testCmdUsage = "test [nodeID]"
	listCmdUsage = "list"
)

func main() {
	// Set log format
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
		os.Exit(1)
	}
}

// execute executes the main program logic
func execute() error {
	cfg := config.ConfigReader
	st, err := service.NewSpeedTest(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize speed test service: %w", err)
	}

	if err := st.Init(); err != nil {
		return fmt.Errorf("failed to initialize speed test environment: %w", err)
	}

	rootCmd := newRootCmd(cfg.Script.Version)
	rootCmd.AddCommand(
		newListCmd(st),
		newTestCmd(st),
	)

	return rootCmd.Execute()
}

// newRootCmd creates the root command
func newRootCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:     "speedtest",
		Short:   programDesc,
		Version: version,
	}
}

// newListCmd creates the list command
func newListCmd(st *service.SpeedTest) *cobra.Command {
	return &cobra.Command{
		Use:   listCmdUsage,
		Short: "List all available nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return st.ListNodes()
		},
	}
}

// newTestCmd creates the test command
func newTestCmd(st *service.SpeedTest) *cobra.Command {
	return &cobra.Command{
		Use:   testCmdUsage,
		Short: "Test the speed of a specific node",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return st.RunAllTest()
			}
			return st.RunTest(args[0])
		},
	}
}
