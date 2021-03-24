package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/github-actions-controller/slack"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var extenderConfig struct {
	appToken string
	botToken string
}

var extenderCmd = &cobra.Command{
	Use:   "extender",
	Short: "extender starts Slack agent to receive interactive messages from Slack",
	Long:  `notifier starts Slack agent to receive interactive messages from Slack`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(extenderConfig.appToken) == 0 {
			log.ErrorExit(errors.New(`"app-token" should not be empty`))
		}

		if len(extenderConfig.botToken) == 0 {
			log.ErrorExit(errors.New(`"bot-token" should not be empty`))
		}
		env := well.NewEnvironment(context.Background())
		s := slack.NewSocketModeClient(
			extenderConfig.appToken,
			extenderConfig.botToken,
		)
		env.Go(s.ListenInteractiveEvents)
		env.Go(s.Run)
		err := well.Wait()
		if err != nil && !well.IsSignaled(err) {
			log.ErrorExit(err)
		}
	},
}

func init() {
	fs := extenderCmd.Flags()
	fs.StringVar(&extenderConfig.appToken, "app-token", "", "The Slack App token.")
	fs.StringVar(&extenderConfig.botToken, "bot-token", "", "The Slack Bot token.")
	rootCmd.AddCommand(extenderCmd)
}
