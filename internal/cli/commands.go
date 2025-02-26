package cli

import (
	"aqua-speed-tools/internal/service"
	"aqua-speed-tools/internal/utils"
	"fmt"

	"github.com/spf13/cobra"
)

// NewListCmd creates the list command
func NewListCmd(st *service.SpeedTest) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return st.ListNodes()
		},
	}
}

// NewTestCmd creates the test command
func NewTestCmd(ts *service.TestService) *cobra.Command {
	return &cobra.Command{
		Use:   "test [nodeID]",
		Short: "Test the speed of a specific node",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return ts.RunAllTest()
			}
			return ts.RunTest(args[0])
		},
	}
}

// ShowLogo displays the program logo
func ShowLogo(repo, version string) {
	logo := `    ___                        _____                     __   ______            __    
   /   | ____ ___  ______ _   / ___/____  ___  ___  ____/ /  /_  __/___  ____  / /____
  / /| |/ __ ` + "`" + `/ / / / __ ` + "`" + `/   \__ \/ __ \/ _ \/ _ \/ __  /    / / / __ \/ __ \/ / ___/
 / ___ / /_/ / /_/ / /_/ /   ___/ / /_/ /  __/  __/ /_/ /    / / / /_/ / /_/ / (__  ) 
/_/  |_\__, /\__,_/\__,_/   /____/ .___/\___/\___/\__,_/    /_/  \____/\____/_/____/  
         /_/                    /_/                                                   `

	fmt.Println(logo)
	utils.Cyan.Printf("\n仓库: https://github.com/%s\n", repo)
	utils.Cyan.Printf("版本: %s\n", version)
	utils.Cyan.Println("作者: Alice39s")
}

// ShowMenu displays the interactive menu
func ShowMenu() {
	utils.Green.Println("请输入要执行选项的数字:")
	fmt.Printf("1) %s列出所有节点%s\n", utils.Bold, utils.Reset)
	fmt.Printf("2) %s测试指定节点%s\n", utils.Bold, utils.Reset)
	fmt.Printf("3) %s退出%s\n", utils.Bold, utils.Reset)
}
