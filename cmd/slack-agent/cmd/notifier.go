package cmd

import (
	"context"
	"errors"

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
	echoBackFlagName   = "echo-back"
)

var notifierConfig struct {
	listenAddr string
	webhookURL string

	doEchoBack bool
}

var notifierCmd = &cobra.Command{
	Use:   "notifier",
	Short: "notifier starts Slack agent to send job results to Slack",
	Long:  `notifier starts Slack agent to send job results to Slack`,
	Run: func(cmd *cobra.Command, args []string) {
		url := viper.GetString(webhookURLFlagName)
		if len(url) == 0 {
			log.ErrorExit(errors.New(`"webhook-url" should not be empty`))
		}

		f := slack.PostWebhook
		if viper.GetBool(echoBackFlagName) {
			f = func(
				webhookURL string,
				msg *slack.WebhookMessage,
			) error {
				cb := slack.InteractionCallback{
					Message: slack.Message{
						Msg: slack.Msg{
							Attachments: msg.Attachments,
						},
					},
				}
				return agent.InteractiveEventHandler(&cb)
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
	fs.StringVar(&notifierConfig.listenAddr, listenAddrFlagName, ":8080", "The address the notifier endpoint binds to")
	fs.StringVarP(&notifierConfig.webhookURL, webhookURLFlagName, "o", "", "The Slack Webhook URL to notify messages to")
	fs.BoolVar(&notifierConfig.doEchoBack, echoBackFlagName, false,
		"This is for development. The notifier annotates a Pod with a deletion time using the extender's WebSocket message handler.",
	)
	rootCmd.AddCommand(notifierCmd)
}
