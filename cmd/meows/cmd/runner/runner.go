package runner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var config struct {
	appID             int64
	appInstallationID int64
	appPrivateKeyPath string
	organizationName  string
}

var githubClient github.Client

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "runner",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			githubClient, err = github.NewClient(
				config.appID,
				config.appInstallationID,
				config.appPrivateKeyPath,
				config.organizationName,
			)
			if err != nil {
				return fmt.Errorf("failed to create github client; %w", err)
			}
			return nil
		},
	}
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newRemoveCmd())

	fs := cmd.PersistentFlags()
	fs.Int64Var(&config.appID, "app-id", 0, "The ID for GitHub App.")
	fs.Int64Var(&config.appInstallationID, "app-installation-id", 0, "The installation ID for GitHub App.")
	fs.StringVar(&config.appPrivateKeyPath, "app-private-key-path", "", "The path for GitHub App private key.")
	fs.StringVarP(&config.organizationName, "organization-name", "o", "", "The GitHub organization name")

	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list REPOSITORY",
		Short: "list runners",
		Long:  "This command lists all runners on the specified repository.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			well.Go(func(ctx context.Context) error {
				runners, err := githubClient.ListRunners(ctx, args[0], nil)
				if err != nil {
					return fmt.Errorf("failed to create github client; %w", err)
				}
				str, err := json.MarshalIndent(runners, "", "    ")
				if err != nil {
					return fmt.Errorf("failed to marshal runners; %w", err)
				}

				fmt.Println(string(str))
				return nil
			})

			well.Stop()
			return well.Wait()
		},
	}
	return cmd
}

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove REPOSITORY",
		Short: "remove offline runners",
		Long:  "This command removes offline runners on the specified repository.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			well.Go(func(ctx context.Context) error {
				runners, err := githubClient.ListRunners(ctx, args[0], nil)
				if err != nil {
					return fmt.Errorf("failed to create github client; %w", err)
				}
				for _, r := range runners {
					if r.Online {
						continue
					}
					err := githubClient.RemoveRunner(ctx, args[0], r.ID)
					if err != nil {
						return fmt.Errorf("failed to remove runner %s (id: %d); %w", r.Name, r.ID, err)
					}
					fmt.Printf("remove runner %s (id: %d)\n", r.Name, r.ID)
				}
				return nil
			})

			well.Stop()
			return well.Wait()
		},
	}
	return cmd
}
