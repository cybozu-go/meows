package slackagent

import (
	"context"

	"github.com/cybozu-go/meows/agent"
	"github.com/spf13/cobra"
)

var config struct {
	server      string
	namespace   string
	jobInfoFile string
	result      string
	channel     string
	extend      bool
	dryRun      bool
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "slackagent",
	}
	cmd.AddCommand(newSendCmd())
	return cmd
}

func newSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send RUNNER_PODNAME [RESULT]",
		Short: "send job result to Slack agent",
		Long: `This command sends job result to Slack agent

For RESULT, specify 'success', 'failure', 'cancelled', or 'unknown'.
If RESULT is omitted or any other value is specified, it will be treated as 'unknown' in the slack-agent server.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			podName := args[0]
			result := ""
			if len(args) > 1 {
				result = args[1]
			}

			var jobInfo *agent.JobInfo
			jobInfo, err := agent.GetJobInfoFromFile(config.jobInfoFile)
			if err != nil {
				return err
			}

			c, err := agent.NewClient(config.server)
			if err != nil {
				return err
			}
			return c.PostResult(context.Background(), config.channel, result, config.extend, config.namespace, podName, jobInfo)
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&config.server, "server", "s", "http://127.0.0.1:8080", "The address to send requests to.")
	fs.StringVarP(&config.namespace, "namespace", "n", "default", "Pod namespace.")
	fs.StringVarP(&config.jobInfoFile, "file", "f", agent.DefaultJobInfoFile, "Job info file.")
	fs.StringVarP(&config.channel, "channel", "c", "", "The Slack channel to notify messages to")
	fs.BoolVarP(&config.extend, "extend", "e", false, "Enable extend button.")

	return cmd
}
