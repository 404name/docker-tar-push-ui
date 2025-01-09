package cmd

import (
	"docker-tar-push-ui/web"

	"github.com/spf13/cobra"
)

var port string

// VersionCmd represents the version command
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "启动web服务",
	Run: func(cmd *cobra.Command, args []string) {
		web.Server(port)
	},
}

func init() {
	ServerCmd.Flags().StringVar(&port, "port", "8088", "server port")
}
