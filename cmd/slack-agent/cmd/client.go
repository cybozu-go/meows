package cmd

import (
	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/spf13/cobra"
)

var clientConfig struct {
	addr      string
	jobName   string
	namespace string
	isFailed  bool
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "client sends requests Slack agent notifier",
	Long:  `client sends requests Slack agent notifier`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := agent.NewNotifierClient(clientConfig.addr)
		if err != nil {
			return err
		}
		return c.PostResult(
			clientConfig.jobName,
			clientConfig.namespace,
			args[0],
			clientConfig.isFailed,
		)
	},
}

func init() {
	fs := clientCmd.Flags()
	fs.StringVarP(&clientConfig.addr, "notifier-address", "a", "127.0.0.1:8080", "The address to send requests to.")
	fs.StringVarP(&clientConfig.namespace, "namespace", "n", "default", "Pod namespace.")
	fs.StringVarP(&clientConfig.jobName, "job", "j", "", "Job name.")
	fs.BoolVar(&clientConfig.isFailed, "failed", false, "Notify the job is failed if enabled.")
	rootCmd.AddCommand(clientCmd)
}
