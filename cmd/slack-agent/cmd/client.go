package cmd

import (
	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/cybozu-go/log"
	"github.com/spf13/cobra"
)

var clientConfig struct {
	addr           string
	workflowName   string
	branchName     string
	repositoryName string
	runID          uint
	namespace      string
	isFailed       bool
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "client sends requests to Slack agent notifier",
	Long:  `client sends requests to Slack agent notifier`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := agent.NewNotifierClient(clientConfig.addr)
		if err != nil {
			log.ErrorExit(err)
		}

		if err := c.PostResult(
			clientConfig.repositoryName,
			clientConfig.workflowName,
			clientConfig.branchName,
			clientConfig.runID,
			clientConfig.namespace,
			args[0],
			clientConfig.isFailed,
		); err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	fs := clientCmd.Flags()
	fs.StringVarP(&clientConfig.addr, "notifier-address", "a", "127.0.0.1:8080", "The address to send requests to.")
	fs.StringVarP(&clientConfig.namespace, "namespace", "n", "default", "Pod namespace.")
	fs.StringVarP(&clientConfig.workflowName, "workflow", "w", "", "Workflow name.")
	fs.StringVarP(&clientConfig.branchName, "branch", "b", "", "Branch name.")
	fs.StringVarP(&clientConfig.repositoryName, "repository", "r", "", "Owner and repository name. (e.g. cybozu-go/github-actions-controller)")
	fs.UintVarP(&clientConfig.runID, "run-id", "i", 0, "Workflow run ID.")
	fs.BoolVar(&clientConfig.isFailed, "failed", false, "Notify the job is failed if enabled.")
	rootCmd.AddCommand(clientCmd)
}
