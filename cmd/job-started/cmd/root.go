package cmd

import (
	"encoding/json"
	"os"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/runner"
	"github.com/spf13/cobra"
)

var (
	jobInfoFile      string
	slackChannelFile string
)

var rootCmd = &cobra.Command{
	Use:  "job-started",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jobInfo, err := runner.GetJobInfo()
		if err != nil {
			return err
		}
		data, err := json.Marshal(jobInfo)
		if err != nil {
			return err
		}
		err = os.WriteFile(jobInfoFile, data, 0664)
		if err != nil {
			return err
		}

		slackChannel := os.Getenv(constants.SlackChannelEnvName)
		return os.WriteFile(slackChannelFile, []byte(slackChannel), 0664)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVarP(&jobInfoFile, "jobinfo-file", "f", constants.RunnerVarDirPath+"/github.env", "Job info file.")
	fs.StringVarP(&slackChannelFile, "slackchannel-file", "s", constants.RunnerVarDirPath+"/slack_channel", "A file that describes the Slack channel to be notified.")
}
