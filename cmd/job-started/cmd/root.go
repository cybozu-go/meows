package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/spf13/cobra"
)

var jobInfoFile string

var rootCmd = &cobra.Command{
	Use:  "job-started",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileName := filepath.Join(os.TempDir(), "started")
		_, err := os.Stat(fileName)
		if os.IsNotExist(err) {
			file, err := os.Create(fileName)
			if err != nil {
				return err
			}
			defer file.Close()
		}
		jobInfo, err := agent.GetJobInfo()
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
	fs.StringVarP(&jobInfoFile, "jobinfo-file", "f", agent.DefaultJobInfoFile, "Job info file.")
}
