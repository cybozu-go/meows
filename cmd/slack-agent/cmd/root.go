package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/cybozu-go/well"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	listenAddrFlagName  = "listen-addr"
	channelFlagName     = "channel"
	appTokenFlagName    = "app-token"
	botTokenFlagName    = "bot-token"
	developmentFlagName = "development"
	verboseFlagName     = "verbose"
)

var rootCmd = &cobra.Command{
	Use:   "slack-agent",
	Short: "Slack agent notifies CI results and accepts requests for extending Pods' lifecycles",
	Long:  `Slack agent notifies CI results and accepts requests for extending Pods' lifecycles`,
	RunE: func(cmd *cobra.Command, args []string) error {
		listenAddr := viper.GetString(listenAddrFlagName)
		devMode := viper.GetBool(developmentFlagName)
		verbose := viper.GetBool(verboseFlagName)
		defaultChannel := viper.GetString(channelFlagName)

		appToken := viper.GetString(appTokenFlagName)
		if len(appToken) == 0 {
			return fmt.Errorf(`"%s" should not be empty`, appTokenFlagName)
		}
		botToken := viper.GetString(botTokenFlagName)
		if len(botToken) == 0 {
			return fmt.Errorf(`"%s" should not be empty`, botTokenFlagName)
		}

		cmd.SilenceUsage = true

		zapLog, err := zap.NewProduction()
		if err != nil {
			return err
		}
		log := zapr.NewLogger(zapLog)

		s := agent.NewServer(log, listenAddr, defaultChannel, appToken, botToken, devMode, verbose)
		well.Go(s.Run)
		well.Stop()
		return well.Wait()
	},
}

func init() {
	fs := rootCmd.Flags()
	fs.String(listenAddrFlagName, ":8080", "The address the notifier endpoint binds to")
	fs.BoolP(developmentFlagName, "d", false, "Development mode.")
	fs.BoolP(verboseFlagName, "v", false, "Verbose.")
	fs.StringP(channelFlagName, "c", "", "The Slack channel to notify messages to")

	fs.String(appTokenFlagName, "", "The Slack App token.")
	fs.String(botTokenFlagName, "", "The Slack Bot token.")

	if err := viper.BindPFlags(fs); err != nil {
		panic(err)
	}
	viper.SetEnvPrefix("slack")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
