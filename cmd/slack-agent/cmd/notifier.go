package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	listenAddrFlagName = "listen-addr"
	webhookURLFlagName = "webhook-url"
)

var notifierCmd = &cobra.Command{
	Use:   "notifier",
	Short: "notifier starts Slack agent to send job results to Slack",
	Long:  `notifier starts Slack agent to send job results to Slack`,
	Run: func(cmd *cobra.Command, args []string) {
		url := viper.GetString(webhookURLFlagName)
		if !isDevelopment && len(url) == 0 {
			log.ErrorExit(errors.New(`"webhook-url" should not be empty`))
		}

		f := slack.PostWebhook
		if isDevelopment {
			f = func(url string, msg *slack.WebhookMessage) error {
				fmt.Printf("development: skip sending message of which text is %q", msg.Text)
				return nil
			}
		}

		env := well.NewEnvironment(context.Background())
		s := agent.NewNotifier(viper.GetString(listenAddrFlagName), url, f)
		env.Go(s.Start)
		err := well.Wait()
		if err != nil && !well.IsSignaled(err) {
			log.ErrorExit(err)
		}
	},
}

func init() {
	fs := notifierCmd.Flags()
	fs.String(listenAddrFlagName, ":8080", "The address the notifier endpoint binds to")
	fs.StringP(webhookURLFlagName, "o", "", "The Slack Webhook URL to notify messages to")
	rootCmd.AddCommand(notifierCmd)
	if err := viper.BindPFlags(fs); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("slack")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
