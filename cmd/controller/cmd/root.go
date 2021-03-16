package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var config struct {
	metricsAddr string
	probeAddr   string

	appID             int64
	appInstallationID int64
	appPrivateKeyPath string
	organizationName  string

	runnerSweepInterval time.Duration
	podSweepInterval    time.Duration
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "github-actions-controller",
	Short: "Kubernetes controller for GitHub Actions self-hosted runner",
	Long:  `Kubernetes controller for GitHub Actions self-hosted runner`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(config.organizationName) == 0 {
			return errors.New("organization-name should be specified")
		}
		if config.appID == 0 {
			return errors.New("app-id should be specified")
		}
		if config.appInstallationID == 0 {
			return errors.New("app-id should be specified")
		}
		if len(config.appPrivateKeyPath) == 0 {
			return errors.New("app-private-key-path should be specified")
		}
		return run()
	},
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
	fs := rootCmd.Flags()
	fs.StringVar(&config.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	fs.StringVar(&config.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	fs.Int64Var(&config.appID, "app-id", 0, "The ID for GitHub App")
	fs.Int64Var(&config.appInstallationID, "app-installation-id", 0, "The installation ID for GitHub App")
	fs.StringVar(&config.appPrivateKeyPath, "app-private-key-path", "", "The path for GitHub App private key")
	fs.StringVarP(&config.organizationName, "organization-name", "o", "", "The GitHub organization name")

	fs.DurationVar(&config.runnerSweepInterval, "runner-sweep-interval", 30*time.Minute, "Interval to watch and sweep unused GitHub Actions runners.")
	fs.DurationVar(&config.podSweepInterval, "pod-sweep-interval", time.Minute, "Interval to watch and delete Pods.")
}
