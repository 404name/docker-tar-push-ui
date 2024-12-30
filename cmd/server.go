package cmd

import (
	"image-upload-portal/web"

	"github.com/spf13/cobra"
)

// VersionCmd represents the version command
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "启动web服务",
	Run: func(cmd *cobra.Command, args []string) {
		web.Server()
	},
}
