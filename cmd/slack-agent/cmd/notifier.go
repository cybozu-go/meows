package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/github-actions-controller/slack"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var notifierConfig struct {
	listenAddr string
	webhookURL string
}

var notifierCmd = &cobra.Command{
	Use:   "notifier",
	Short: "notifier starts Slack agent to send job results to Slack",
	Long:  `notifier starts Slack agent to send job results to Slack`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(notifierConfig.webhookURL) == 0 {
			log.ErrorExit(errors.New(`"webhook-url" should not be empty`))
		}

		env := well.NewEnvironment(context.Background())
		s := slack.NewNotifier(
			notifierConfig.listenAddr,
			slack.NewWebhookClient(notifierConfig.webhookURL),
		)
		env.Go(s.Start)
		err := well.Wait()
		if err != nil && !well.IsSignaled(err) {
			log.ErrorExit(err)
		}
	},
}

func init() {
	fs := notifierCmd.Flags()
	fs.StringVar(&notifierConfig.listenAddr, "listen-addr", ":8080", "The address the notifier endpoint binds to")
	fs.StringVarP(&notifierConfig.webhookURL, "webhook-url", "o", "", "The Slack Webhook URL to notify messages to")
	rootCmd.AddCommand(notifierCmd)
}
