package main

import (
	"aqua-speed-tools/internal/cli"
	"aqua-speed-tools/internal/config"
	"aqua-speed-tools/internal/service"
	"aqua-speed-tools/internal/updater"
	"aqua-speed-tools/internal/utils"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	version = "3.0.1"
	repo    = "alice39s/aqua-speed-tools"
)

var (
	// Flags
	githubRawMagicURL string
	githubAPIMagicURL string
	dohEndpoint       string
	debugMode         bool
	useMirrors        bool

	// Services
	st     *service.SpeedTest
	ts     *service.TestService
	logger *zap.Logger
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
		os.Exit(1)
	}
}

// execute executes the main program logic
func execute() error {
	// 设置调试模式并初始化日志
	utils.IsDebug = debugMode
	utils.ResetLogger()

	// 初始化配置
	if err := initConfig(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// 初始化服务
	if err := initServices(); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// 执行命令
	rootCmd := newRootCmd(config.ConfigReader.Script.Version)
	return rootCmd.Execute()
}

// initConfig initializes the configuration
func initConfig() error {
	// 首先加载配置文件
	if err := config.LoadConfig(""); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.ConfigReader

	// 如果启用镜像模式，使用配置文件中的镜像设置
	if useMirrors {
		utils.Info("正在使用 GitHub 镜像模式")

		// 设置 API 镜像
		if githubAPIMagicURL != "" {
			cfg.GithubAPIBaseURL = githubAPIMagicURL
			utils.Debug("使用命令行指定的 API 镜像",
				zap.String("url", githubAPIMagicURL))
		} else if cfg.GithubAPIMagicURL != "" {
			cfg.GithubAPIBaseURL = cfg.GithubAPIMagicURL
			utils.Debug("使用配置文件中的 API 镜像",
				zap.String("url", cfg.GithubAPIMagicURL))
		}

		// 测试并选择最快的 Raw 镜像
		if len(cfg.GithubRawJsdelivrSet) > 0 {
			mirrorTester := service.NewMirrorTester(utils.GetLogger(), 5*time.Second)
			fastestMirror := mirrorTester.FindFastestMirror(cfg.GithubRawJsdelivrSet)

			if fastestMirror != "" {
				githubRawMagicURL = fastestMirror
				cfg.GithubRawBaseURL = githubRawMagicURL
				utils.Info("使用最快的 Raw 镜像",
					zap.String("url", githubRawMagicURL))
			} else {
				utils.Warning("所有镜像都不可用，使用默认 GitHub URL")
			}
		}
	}

	// 确保基础 URL 不为空
	if cfg.GithubAPIBaseURL == "" {
		cfg.GithubAPIBaseURL = "https://api.github.com"
		utils.Debug("使用默认 API URL", zap.String("url", cfg.GithubAPIBaseURL))
	}
	if cfg.GithubRawBaseURL == "" {
		cfg.GithubRawBaseURL = "https://raw.githubusercontent.com"
		utils.Debug("使用默认 Raw URL", zap.String("url", cfg.GithubRawBaseURL))
	}

	// 输出调试信息
	if debugMode {
		utils.Debug("配置信息",
			zap.String("版本", version),
			zap.String("仓库", repo),
			zap.String("GitHub API URL", cfg.GithubAPIBaseURL),
			zap.String("GitHub Raw URL", cfg.GithubRawBaseURL),
			zap.String("GitHub API Magic URL", cfg.GithubAPIMagicURL),
			zap.Any("GitHub Raw jsDelivr Set", cfg.GithubRawJsdelivrSet),
			zap.Any("DNS over HTTPS Set", cfg.DNSOverHTTPSSet),
			zap.Int("下载超时时间", cfg.DownloadTimeout),
			zap.String("日志级别", cfg.LogLevel))
	}

	return nil
}

// initServices initializes all required services
func initServices() error {
	cfg := config.ConfigReader

	// 初始化 DNS 解析器
	if err := initDNSResolver(); err != nil {
		return err
	}

	// 初始化更新器
	urls := utils.NewGitHubURLs(
		cfg.GithubRawBaseURL,
		cfg.GithubAPIBaseURL,
		cfg.GithubRawJsdelivrSet,
	)
	updater, err := updater.NewWithLocalVersionAndURLs(version, urls)
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	// 初始化速度测试服务
	st, err = service.NewSpeedTest(*cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize speed test service: %w", err)
	}

	if err := st.Init(); err != nil {
		return fmt.Errorf("failed to initialize speed test environment: %w", err)
	}

	// 初始化测试服务
	ts = service.NewTestService(st.GetNodes(), utils.GetLogger(), updater)

	return nil
}

// initDNSResolver initializes the DNS resolver
func initDNSResolver() error {
	if dohEndpoint != "" {
		// 使用命令行指定的 DoH 端点
		utils.Debug("使用命令行指定的 DoH 端点", zap.String("endpoint", dohEndpoint))
		resolver, err := utils.NewDNSResolver(dohEndpoint, 10, 3)
		if err != nil {
			return fmt.Errorf("failed to initialize DNS resolver: %w", err)
		}
		utils.SetDNSResolver(resolver)
	} else if len(config.ConfigReader.DNSOverHTTPSSet) > 0 {
		// 使用配置文件中的第一个 DoH 端点
		doh := config.ConfigReader.DNSOverHTTPSSet[0]
		utils.Debug("使用配置文件中的 DoH 端点",
			zap.String("endpoint", doh.Endpoint),
			zap.Int("timeout", doh.Timeout),
			zap.Int("retries", doh.Retries))
		resolver, err := utils.NewDNSResolver(doh.Endpoint, doh.Timeout, doh.Retries)
		if err != nil {
			return fmt.Errorf("failed to initialize DNS resolver: %w", err)
		}
		utils.SetDNSResolver(resolver)
	}

	return nil
}

// newRootCmd creates the root command
func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "aqua-speed-tools",
		Short:   "Network Speed Test Tool - Supports testing network speed for specific nodes or all nodes",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 默认进入交互模式
			return runInteractiveMode()
		},
	}

	// Add flags
	cmd.PersistentFlags().StringVar(&githubRawMagicURL, "github-raw-magic-url", "", "设置 GitHub Raw Magic URL")
	cmd.PersistentFlags().StringVar(&githubAPIMagicURL, "github-api-magic-url", "", "设置 GitHub API Magic URL")
	cmd.PersistentFlags().StringVar(&dohEndpoint, "doh-endpoint", "", "设置 DNS over HTTPS 端点")
	cmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "开启调试模式")
	cmd.PersistentFlags().BoolVar(&useMirrors, "use-mirrors", false, "启用配置文件中的镜像设置")

	return cmd
}

// runInteractiveMode runs the interactive mode
func runInteractiveMode() error {
	cli.ShowLogo(repo, version)
	for {
		cli.ShowMenu()
		var choice int
		fmt.Scanf("%d", &choice)

		switch choice {
		case 1:
			utils.Blue.Println("列出所有节点...")
			if err := st.ListNodes(); err != nil {
				utils.Red.Printf("列出节点失败: %v\n", err)
				continue
			}
		case 2:
			utils.Blue.Print("请输入节点 ID (支持数字序号或英文ID): ")
			var nodeID string
			fmt.Scanf("%s", &nodeID)

			if err := ts.RunTest(nodeID); err != nil {
				utils.Red.Printf("测试节点失败: %v\n", err)
				continue
			}
		case 3:
			utils.Yellow.Println("正在退出...")
			return nil
		default:
			utils.Red.Println("无效选项，请重新输入")
		}
	}
}
