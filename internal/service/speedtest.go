package service

import (
	"aqua-speed-tools/internal/config"
	"aqua-speed-tools/internal/models"
	"aqua-speed-tools/internal/updater"
	"fmt"
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
