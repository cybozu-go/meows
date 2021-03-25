package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "Slack agent for GitHub Actions self-hosted runner",
	Short: "Slack agent notifies CI results via Webhook and accepts requests for extending Pods' lifecycles",
	Long:  `Slack agent notifies CI results via Webhook and accepts requests for extending Pods' lifecycles`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	viper.SetEnvPrefix("slack")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
