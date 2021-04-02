package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	appTokenFlagName = "app-token"
	botTokenFlagName = "bot-token"
	retryFlagName    = "retry"
)

var extenderCmd = &cobra.Command{
	Use:   "extender",
	Short: "extender starts Slack agent to receive interactive messages from Slack",
	Long:  `notifier starts Slack agent to receive interactive messages from Slack`,
	Run: func(cmd *cobra.Command, args []string) {
		appToken := viper.GetString(appTokenFlagName)
		if len(appToken) == 0 {
			log.ErrorExit(fmt.Errorf(`"%s" should not be empty`, appTokenFlagName))
		}

		botToken := viper.GetString(botTokenFlagName)
		if len(botToken) == 0 {
			log.ErrorExit(fmt.Errorf(`"%s" should not be empty`, botTokenFlagName))
		}

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

		var err error
		s := agent.NewSocketModeClient(appToken, botToken, a)
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
			err = well.Wait()
			log.Warn("failed to open a connection with Slack", map[string]interface{}{
				"trycount": i + 1,
				"retry":    retry,
			})
			time.Sleep(time.Minute)
		}
		if err != nil && !well.IsSignaled(err) {
			log.ErrorExit(err)
		}
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

	viper.SetEnvPrefix("slack")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
