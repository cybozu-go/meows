package slackagent

import (
	"context"
	"fmt"
	"os"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/agent"
	"github.com/cybozu-go/meows/runner"
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

var slackChannelFilePath string

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "slackagent",
	}
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newSetChannelCmd())
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

			var jobInfo *runner.JobInfo
			jobInfo, err := runner.GetJobInfoFromFile(config.jobInfoFile)
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
	fs.StringVarP(&config.jobInfoFile, "file", "f", constants.RunnerVarDirPath+"/github.env", "Job info file.")
	fs.StringVarP(&config.channel, "channel", "c", "", "The Slack channel to notify messages to")
	fs.BoolVarP(&config.extend, "extend", "e", false, "Enable extend button.")

	return cmd
}

func newSetChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-channel [SLACK_CHANNEL_NAME]",
		Short: "set Slack channel to notify.",
		Long: `This command set a Slack channel to notified job result.
This is used by calling it in the workflow yaml file.
If SLACK_CHANNEL_NAME is not specified, the environment variable MEOWS_SLACK_CHANNEL is specified as a Slack channel to notified.`,

		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var channel string
			if len(args) == 1 {
				channel = args[0]
			}

			if channel == "" {
				channel = os.Getenv(constants.SlackChannelEnvName)
			}

			err := os.WriteFile(slackChannelFilePath, []byte(channel), 0644)
			if err != nil {
				return err
			}
			fmt.Printf("Set slack channel to '%s'.\n", channel)
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&slackChannelFilePath, "file", "f", constants.SlackChannelFilePath, "A file that describes the Slack channel to be notified.")
	return cmd
}
