package cmd

import (
	"errors"
	"flag"
	"os"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const defaultRunnerImage = "ghcr.io/cybozu-go/meows-runner:latest"

var config struct {
	zapOpts               zap.Options
	metricsAddr           string
	probeAddr             string
	webhookAddr           string
	controllerNamespace   string
	runnerImage           string
	runnerManagerInterval time.Duration
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "controller",
	Short: "Kubernetes controller for GitHub Actions self-hosted runner",
	Long:  `Kubernetes controller for GitHub Actions self-hosted runner`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.controllerNamespace = os.Getenv(constants.PodNamespaceEnvName)
		if config.controllerNamespace == "" {
			return errors.New(constants.PodNamespaceEnvName + " should be passed")
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
	fs.StringVar(&config.runnerImage, "runner-image", defaultRunnerImage, "The image of runner container")
	fs.DurationVar(&config.runnerManagerInterval, "runner-manager-interval", time.Minute, "Interval to watch and delete Pods.")

	goflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(goflags)
	config.zapOpts.BindFlags(goflags)
	fs.AddGoFlagSet(goflags)
}
