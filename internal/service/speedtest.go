package service

import (
	"aqua-speed-tools/internal/config"
	"aqua-speed-tools/internal/models"
	"aqua-speed-tools/internal/updater"
	"fmt"

	"go.uber.org/zap"
)

// SpeedTest provides network speed testing functionality
type SpeedTest struct {
	config  config.Config    // Configuration information
	nodes   models.NodeList  // Node list
	updater *updater.Updater // Updater
	logger  *zap.Logger      // Logger
}

// NewSpeedTest creates a new SpeedTest instance
func NewSpeedTest(cfg config.Config) (*SpeedTest, error) {
	updater, err := updater.NewWithLocalVersion("0.0.0") // Start with 0.0.0 version, will be updated by GitHub API
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &SpeedTest{
		config:  cfg,
		nodes:   make(models.NodeList),
		updater: updater,
		logger:  logger,
	}, nil
}

// Init initializes the speed test environment
func (s *SpeedTest) Init() error {
	// 检查更新
	if err := s.updater.CheckAndUpdate(); err != nil {
		s.logger.Error("Failed to check for updates", zap.Error(err))
		// 继续执行，不要因为更新检查失败而中断
	}

	// Initialize nodes
	return s.initNodes()
}

func (s *SpeedTest) GetNodes() []models.Node {
	nodes := make([]models.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetUpdater returns the updater instance
func (s *SpeedTest) GetUpdater() *updater.Updater {
	return s.updater
}
