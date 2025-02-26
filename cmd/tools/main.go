package main

import (
	"aqua-speed-tools/internal/cli"
	"aqua-speed-tools/internal/config"
	"aqua-speed-tools/internal/service"
	"aqua-speed-tools/internal/updater"
	"aqua-speed-tools/internal/utils"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	version = "3.0.0"
	repo    = "alice39s/aqua-speed-tools"
)

var (
	// Flags
	githubRawMagicURL string
	githubAPIMagicURL string
	dohEndpoint       string
	debugMode         bool

	// Services
	st     *service.SpeedTest
	ts     *service.TestService
	logger *zap.Logger
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

	// 设置调试模式
	utils.IsDebug = debugMode

	// 初始化 DNS 解析器
	if dohEndpoint != "" {
		// 使用命令行指定的 DoH 端点
		resolver, err := utils.NewDNSResolver(dohEndpoint, 10, 3)
		if err != nil {
			return fmt.Errorf("failed to initialize DNS resolver: %w", err)
		}
		utils.SetDNSResolver(resolver)
	} else if len(cfg.DNSOverHTTPSSet) > 0 {
		// 使用配置文件中的第一个 DoH 端点
		doh := cfg.DNSOverHTTPSSet[0]
		resolver, err := utils.NewDNSResolver(doh.Endpoint, doh.Timeout, doh.Retries)
		if err != nil {
			return fmt.Errorf("failed to initialize DNS resolver: %w", err)
		}
		utils.SetDNSResolver(resolver)
	}

	// 设置 GitHub URLs
	urls := utils.NewGitHubURLs(githubRawMagicURL, githubAPIMagicURL, cfg.GitHubRawMagicSet)
	cfg.GithubRawBaseUrl = urls.RawBaseURL
	cfg.GithubApiBaseUrl = urls.APIURL

	// 初始化服务
	var err error
	logger = updater.InitLogger()
	updater, err := updater.NewWithLocalVersion(version)
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	st, err = service.NewSpeedTest(*cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize speed test service: %w", err)
	}

	if err := st.Init(); err != nil {
		return fmt.Errorf("failed to initialize speed test environment: %w", err)
	}

	ts = service.NewTestService(st.GetNodes(), logger, updater)

	rootCmd := newRootCmd(cfg.Script.Version)
	rootCmd.AddCommand(
		cli.NewListCmd(st),
		cli.NewTestCmd(ts),
	)

	return rootCmd.Execute()
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
