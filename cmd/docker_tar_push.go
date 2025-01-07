package cmd

import (
	"image-upload-portal/pkg/push"

	"github.com/silenceper/log"
	"github.com/spf13/cobra"
)

var (
	registryURL   string
	username      string
	password      string
	imagePrefix   string
	skipSSLVerify bool
	logLevel      int

	DockerTarPushCmd = &cobra.Command{
		Use:   "docker-tar-push",
		Short: "push your docker tar archive image without docker",
		Long:  `push your docker tar archive image without docker.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log.SetLogLevel(log.Level(logLevel))
			imagePush := push.NewImagePush(args[0], registryURL, imagePrefix, username, password, skipSSLVerify, nil)
			imagePush.Push()
		},
	}
)

func init() {
	DockerTarPushCmd.Flags().StringVar(&registryURL, "registry", "", "registry url")
	DockerTarPushCmd.Flags().StringVar(&username, "username", "", "registry auth username")
	DockerTarPushCmd.Flags().StringVar(&password, "password", "", "registry auth password")
	DockerTarPushCmd.Flags().StringVar(&imagePrefix, "image-prefix", "", "add image repo prefix")
	DockerTarPushCmd.Flags().BoolVar(&skipSSLVerify, "skip-ssl-verify", true, "skip ssl verify")
	DockerTarPushCmd.Flags().IntVar(&logLevel, "log-level", log.LevelInfo, "log-level, 0:Fatal,1:Error,2:Warn,3:Info,4:Debug")

	DockerTarPushCmd.MarkFlagRequired("registry")
}
