package cmd

import (
	"encoding/json"
	"io"
	"os"

	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/spf13/cobra"
)

var clientConfig struct {
	server      string
	namespace   string
	jobInfoFile string
	result      string
	channel     string
	extend      bool
	dryRun      bool
}

var rootCmd = &cobra.Command{
	Use:   "slack-agent-client RUNNER_PODNAME [RESULT]",
	Short: "slack-agent-client sends job result to Slack agent",
	Long: `slack-agent-client sends job result to Slack agent

For RESULT, specify 'success', 'failure', 'cancelled', or 'unknown'.
If RESULT is omitted or any other value is specified, it will be treated as 'unknown' in the slack-agent server.
`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		podName := args[0]
		result := ""
		if len(args) > 1 {
			result = args[1]
		}

		var data []byte
		if clientConfig.jobInfoFile == "-" {
			d, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			data = d
		} else {
			d, err := os.ReadFile(clientConfig.jobInfoFile)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			// Accepts ErrNotExist.
			// When ErrNotExist is returned, it means that `job_started` was not called in a workflow.
			// Even in this case, slack notification will run.
			data = d
		}

		var jobInfo *agent.JobInfo
		if len(data) != 0 {
			tmp := &agent.JobInfo{}
			err := json.Unmarshal(data, tmp)
			if err != nil {
				return err
			}
			jobInfo = tmp
		}

		c, err := agent.NewClient(clientConfig.server)
		if err != nil {
			return err
		}
		return c.PostResult(clientConfig.channel, result, clientConfig.extend, clientConfig.namespace, podName, jobInfo)
	},
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVarP(&clientConfig.server, "server", "s", "http://127.0.0.1:8080", "The address to send requests to.")
	fs.StringVarP(&clientConfig.namespace, "namespace", "n", "default", "Pod namespace.")
	fs.StringVarP(&clientConfig.jobInfoFile, "file", "f", agent.DefaultJobInfoFile, "Job info file.")
	fs.StringVarP(&clientConfig.channel, "channel", "c", "", "The Slack channel to notify messages to")
	fs.BoolVarP(&clientConfig.extend, "extend", "e", false, "Enable extend button.")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
