package cmd

import (
	"context"
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
	appTokenFlagName = "app-token"
	botTokenFlagName = "bot-token"
	retryFlagName    = "retry"
	noExtendFlagName = "no-extend"
)

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

		f := agent.InteractiveEventHandler
		if viper.GetBool(noExtendFlagName) {
			f = func(cb *slack.InteractionCallback) error {
				fmt.Println(cb.Message.Text)
				return nil
			}
		}

		s := agent.NewSocketModeClient(appToken, botToken, f)

		var err error
		retry := viper.GetUint("retry")
		for i := uint(0); i < retry+1; i++ {
			env := well.NewEnvironment(context.Background())
			env.Go(func(_ context.Context) error {
				return s.ListenInteractiveEvents()
			})
			env.Go(func(_ context.Context) error {
				return s.Run()
			})
			err = well.Wait()
			log.Warn("failed to open a connection with Slack", map[string]interface{}{
				"trycount": i + 1,
				"retry":    retry,
			})
		}
		return err
	},
}

func init() {
	fs := extenderCmd.Flags()
	fs.String(appTokenFlagName, "", "The Slack App token.")
	fs.String(botTokenFlagName, "", "The Slack Bot token.")
	fs.Uint(retryFlagName, 0, "How many times the extender retries to connect Slack.")
	fs.BoolP(noExtendFlagName, "d", false,
		"The extender just writes messages to stdout when receiving message.",
	)
	rootCmd.AddCommand(extenderCmd)
	if err := viper.BindPFlags(fs); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("slack")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
