package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type MirrorTester struct {
	client  *http.Client
	logger  *zap.Logger
	timeout time.Duration
}

type MirrorResult struct {
	URL       string
	Latency   time.Duration
	Reachable bool
}

func NewMirrorTester(logger *zap.Logger, timeout time.Duration) *MirrorTester {
	return &MirrorTester{
		client: &http.Client{
			Timeout: timeout,
		},
		logger:  logger,
		timeout: timeout,
	}
}

func (m *MirrorTester) testSingleMirror(ctx context.Context, mirrorURL string) MirrorResult {
	result := MirrorResult{
		URL:       mirrorURL,
		Reachable: false,
		Latency:   time.Hour,
	}

	testURL := fmt.Sprintf("%s/alice39s/aqua-speed@main/README.md",
		strings.TrimSuffix(mirrorURL, "/"))

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		m.logger.Debug("创建请求失败", zap.String("url", mirrorURL), zap.Error(err))
		return result
	}

	resp, err := m.client.Do(req)
	if err != nil {
		m.logger.Debug("请求失败", zap.String("url", mirrorURL), zap.Error(err))
		return result
	}
	defer resp.Body.Close()

	result.Latency = time.Since(start)
	result.Reachable = true

	return result
}

func (m *MirrorTester) FindFastestMirror(mirrors []string) string {
	if len(mirrors) == 0 {
		return ""
	}

	var bestMirror string
	var bestLatency time.Duration = time.Hour

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	for _, mirror := range mirrors {
		var totalLatency time.Duration
		reachableCount := 0

		for i := 0; i < 2; i++ {
			result := m.testSingleMirror(ctx, mirror)
			if result.Reachable {
				totalLatency += result.Latency
				reachableCount++
			}
		}

		if reachableCount > 0 {
			avgLatency := totalLatency / time.Duration(reachableCount)
			m.logger.Debug("镜像测试结果",
				zap.String("mirror", mirror),
				zap.Duration("avgLatency", avgLatency),
				zap.Int("reachableCount", reachableCount))

			if avgLatency < bestLatency {
				bestLatency = avgLatency
				bestMirror = mirror
			}
		}
	}

	if bestMirror != "" {
		m.logger.Info("找到最快的镜像",
			zap.String("mirror", bestMirror),
			zap.Duration("latency", bestLatency))
	}

	return bestMirror
}
