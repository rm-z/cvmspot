package cli

import (
	"cvmspot/tcloud"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	listFlag   bool
	deleteFlag bool
	client     *tcloud.Client
)

var rootCmd = &cobra.Command{
	Use:   "cvmspot",
	Short: "腾讯云CVM竞价实例管理工具",
	Long:  `用于管理腾讯云CVM竞价实例的命令行工具`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() // 默认显示帮助
	},
}

var cvmCmd = &cobra.Command{
	Use:   "cvm",
	Short: "CVM实例管理",
	Long:  `管理腾讯云CVM实例的命令`,
	Run: func(cmd *cobra.Command, args []string) {
		if listFlag {
			client.ListInstances()
			return
		}
		if deleteFlag {
			if len(args) == 0 {
				fmt.Println("错误：删除操作需要至少一个实例ID（例如：cvmspot cvm -d i-123456 i-789012）")
				os.Exit(1)
			}
			client.DeleteInstances(args) // 传入所有参数作为实例ID
			return
		}
		cmd.Help()
	},
}

func Execute(c *tcloud.Client) {
	client = c
	rootCmd.AddCommand(cvmCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cvmCmd.Flags().BoolVarP(&listFlag, "list", "l", false, "列出所有运行中的实例")
	cvmCmd.Flags().BoolVarP(&deleteFlag, "delete", "d", false, "删除指定的实例（支持多个ID，用空格分隔）")
}
