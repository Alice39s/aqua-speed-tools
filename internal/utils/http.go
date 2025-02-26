package utils

import (
	"fmt"
	"net/http"
	"time"
)

const maxAttempts = 3
const connectTimeout = 10 * time.Second
const maxTime = 30 * time.Second
const apiMaxTime = 15 * time.Second

// HttpGet 发送 HTTP GET 请求
func HttpGet(url string) (*http.Response, error) {
	attempt := 1

	for attempt <= maxAttempts {
		LogDebug("正在请求 %s", url)

		client := &http.Client{
			Timeout: maxTime,
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %w", err)
		}

		req.Header.Set("User-Agent", "aqua-speed-tools/1.0.0")

		resp, err := client.Do(req)
		if err != nil {
			LogWarning("请求失败，尝试第 %d/%d 次", attempt, maxAttempts)
			attempt++
			if attempt <= maxAttempts {
				time.Sleep(2 * time.Second)
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			LogWarning("请求失败，尝试第 %d/%d 次", attempt, maxAttempts)
			attempt++
			if attempt <= maxAttempts {
				time.Sleep(2 * time.Second)
			}
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("请求失败: %s", url)
}
