package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/spf13/cobra"
)

var config struct {
	namespace string
	after     time.Duration
}

var rootCmd = &cobra.Command{
	Use:   "Deletion time annotator",
	Short: "Annotate deletion time from inside Pods",
	Long:  `Annotate deletion time from inside Pods`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.AnnotateDeletionTime(
			args[0],
			config.namespace,
			time.Now().UTC().Add(config.after),
		)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVarP(&config.namespace, "namespace", "n", "default", "Pod namespace.")
	fs.DurationVarP(&config.after, "after", "a", 0,
		"Annotate Pod with the time after the given duration.",
	)
}
