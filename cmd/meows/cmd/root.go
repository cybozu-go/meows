package cmd

import (
	"os"

	"github.com/cybozu-go/meows/cmd/meows/cmd/runner"
	"github.com/cybozu-go/meows/cmd/meows/cmd/slackagent"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "meows",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(slackagent.NewCommand())
	rootCmd.AddCommand(runner.NewCommand())
}
