package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   string
	flag      bool
	staticDir string
)

var RootCmd = &cobra.Command{
	Use:   "cobra",
	Short: "一个命令行cli脚手架",
	Long: `A Commander for modern Go CLI interactions，
	一个命令行cli脚手架`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func init() {
	RootCmd.PersistentFlags().StringVar(&version, "version", "1.0.0", "测试一下")
	RootCmd.PersistentFlags().BoolVar(&flag, "debug", false, "start with debug mode")
	RootCmd.PersistentFlags().StringVar(&staticDir, "static", "./public/dist", "静态文件夹路径")
	RootCmd.AddCommand(VersionCmd)
	RootCmd.AddCommand(ServerCmd)
	// 在RootCmd Excute前，version这些都还只是初始值
}

// OutAlistInit 暴露用于外部启动server的函数
func OutAlistInit() {
	var (
		cmd  *cobra.Command
		args []string
	)
	ServerCmd.Run(cmd, args)
}
