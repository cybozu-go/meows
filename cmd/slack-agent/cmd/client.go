package cmd

import (
	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/spf13/cobra"
)

var clientConfig struct {
	addr             string
	workflowName     string
	branchName       string
	repositoryName   string
	organizationName string
	runID            uint
	namespace        string
	isFailed         bool
}

var clientCmd = &cobra.Command{
	Use:   "client PODNAME",
	Short: "client sends requests to Slack agent notifier",
	Long:  `client sends requests to Slack agent notifier`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		c, err := agent.NewNotifierClient(clientConfig.addr)
		if err != nil {
			return err
		}

		return c.PostResult(
			clientConfig.repositoryName,
			clientConfig.organizationName,
			clientConfig.workflowName,
			clientConfig.branchName,
			clientConfig.runID,
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
	fs.StringVarP(&clientConfig.workflowName, "workflow", "w", "", "Workflow name.")
	fs.StringVarP(&clientConfig.branchName, "branch", "b", "", "Branch name.")
	fs.StringVarP(&clientConfig.organizationName, "organization", "o", "", "Organization name.")
	fs.StringVarP(&clientConfig.repositoryName, "repository", "r", "", "Repository name.")
	fs.UintVarP(&clientConfig.runID, "run-id", "i", 0, "Workflow run ID.")
	fs.BoolVar(&clientConfig.isFailed, "failed", false, "Notify the job is failed if enabled.")
	rootCmd.AddCommand(clientCmd)
}
