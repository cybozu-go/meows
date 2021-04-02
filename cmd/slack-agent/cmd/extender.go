package cmd

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/cybozu-go/well"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/klog"
)

const (
	appTokenFlagName = "app-token"
	botTokenFlagName = "bot-token"
	retryFlagName    = "retry"
)

var zapOpts zap.Options

var extenderCmd = &cobra.Command{
	Use:   "extender",
	Short: "extender starts Slack agent to receive interactive messages from Slack",
	Long:  `notifier starts Slack agent to receive interactive messages from Slack`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appToken := viper.GetString(appTokenFlagName)
		if len(appToken) == 0 {
			return fmt.Errorf(`"%s" should not be empty`, appTokenFlagName)
		}

		botToken := viper.GetString(botTokenFlagName)
		if len(botToken) == 0 {
			return fmt.Errorf(`"%s" should not be empty`, botTokenFlagName)
		}

		cmd.SilenceUsage = true

		zapLog, err := zap.NewDevelopment(zapOpts)
		if err != nil {
			return err
		}
		log := zapr.NewLogger(zapLog)

		a := agent.AnnotateDeletionTime
		if isDevelopment {
			a = func(_ context.Context, name string, namespace string, t time.Time) error {
				fmt.Printf(
					"development: annotated %s to %s in %s",
					t.UTC().Format(time.RFC3339),
					name,
					namespace,
				)
				return nil
			}
		}

		s := agent.NewSocketModeClient(&log, appToken, botToken, a)
		// retry every 1 minute if failed to open connection
		// because rate limit for `connection.open` is so small.
		// https://api.slack.com/methods/apps.connections.open
		retry := viper.GetUint("retry")
		for i := uint(0); i < retry+1; i++ {
			env := well.NewEnvironment(context.Background())
			env.Go(s.ListenInteractiveEvents)
			env.Go(func(_ context.Context) error {
				return s.Run()
			})
			err := well.Wait()
			if i == retry && err != nil {
				return err
			}
			log.Info("retry opening a connection with Slack", "trycount", i+1, "retry", retry)
			time.Sleep(time.Minute)
		}
		return nil
	},
}

func init() {
	fs := extenderCmd.Flags()
	fs.String(appTokenFlagName, "", "The Slack App token.")
	fs.String(botTokenFlagName, "", "The Slack Bot token.")
	fs.Uint(retryFlagName, 0, "How many times the extender retries to connect Slack.")
	rootCmd.AddCommand(extenderCmd)
	if err := viper.BindPFlags(fs); err != nil {
		panic(err)
	}

	goflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(goflags)
	zapOpts.BindFlags(goflags)

	fs.AddGoFlagSet(goflags)

	viper.SetEnvPrefix("slack")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
