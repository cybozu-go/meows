package cmd

import (
	"errors"
	"flag"
	"os"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const defaultRunnerImage = "quay.io/cybozu/meows-runner:" + constants.Version

var config struct {
	zapOpts zap.Options

	metricsAddr string
	probeAddr   string
	webhookAddr string

	appID             int64
	appInstallationID int64
	appPrivateKeyPath string
	organizationName  string

	runnerImage           string
	runnerManagerInterval time.Duration
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "controller",
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

		// Prevent to show the version string in the help, to keep consistent the output of the command and the documentation.
		// Please fix this when the documentation is revised.
		if len(config.runnerImage) == 0 {
			config.runnerImage = defaultRunnerImage
		}
		return run()
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
	fs.StringVar(&config.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	fs.StringVar(&config.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.StringVar(&config.webhookAddr, "webhook-addr", ":9443", "The address the webhook endpoint binds to")

	fs.Int64Var(&config.appID, "app-id", 0, "The ID for GitHub App.")
	fs.Int64Var(&config.appInstallationID, "app-installation-id", 0, "The installation ID for GitHub App.")
	fs.StringVar(&config.appPrivateKeyPath, "app-private-key-path", "", "The path for GitHub App private key.")
	fs.StringVarP(&config.organizationName, "organization-name", "o", "", "The GitHub organization name")

	fs.StringVar(&config.runnerImage, "runner-image", "", "The image of runner container")
	fs.DurationVar(&config.runnerManagerInterval, "runner-manager-interval", time.Minute, "Interval to watch and delete Pods.")

	goflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(goflags)
	config.zapOpts.BindFlags(goflags)
	fs.AddGoFlagSet(goflags)
}
