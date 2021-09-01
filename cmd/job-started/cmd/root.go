package cmd

import (
	"encoding/json"
	"os"

	"github.com/cybozu-go/meows/runner/client"
	"github.com/spf13/cobra"
)

var jobInfoFile string

var rootCmd = &cobra.Command{
	Use:  "job-started",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jobInfo, err := client.GetJobInfo()
		if err != nil {
			return err
		}
		data, err := json.Marshal(jobInfo)
		if err != nil {
			return err
		}
		return os.WriteFile(jobInfoFile, data, 0664)
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
	fs.StringVarP(&jobInfoFile, "jobinfo-file", "f", client.DefaultJobInfoFile, "Job info file.")
}
