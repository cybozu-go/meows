package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var config struct {
	appID               int64
	appInstallationID   int64
	appPrivateKeyPath   string
	personalAccessToken string
}

var githubClient github.Client

func splitOwnerRepo(str string) (string, string, error) {
	split := strings.Split(str, "/")
	switch len(split) {
	case 1:
		return split[0], "", nil
	case 2:
		return split[0], split[1], nil
	default:
		return "", "", fmt.Errorf("invalid format: %s", str)
	}
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "runner",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var cred *github.ClientCredential
			if config.personalAccessToken != "" {
				cred = &github.ClientCredential{
					PersonalAccessToken: config.personalAccessToken,
				}
			} else {
				cred = &github.ClientCredential{
					AppID:             config.appID,
					AppInstallationID: config.appInstallationID,
					PrivateKeyPath:    config.appPrivateKeyPath,
				}
			}

			var err error
			githubClient, err = github.NewFactory().New(cred)
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
	fs.StringVar(&config.personalAccessToken, "token", "", "The personal access token (PAT) of GitHub.")
	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [ORGANIZATION | REPOSITORY]",
		Short: "list runners",
		Long:  "This command lists all runners on the specified organization or repository.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			owner, repo, err := splitOwnerRepo(args[0])
			if err != nil {
				return err
			}

			well.Go(func(ctx context.Context) error {
				runners, err := githubClient.ListRunners(ctx, owner, repo, nil)
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
		Use:   "remove  [ORGANIZATION | REPOSITORY]",
		Short: "remove offline runners",
		Long:  "This command removes offline runners on the specified organization or repository.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			owner, repo, err := splitOwnerRepo(args[0])
			if err != nil {
				return err
			}

			well.Go(func(ctx context.Context) error {
				runners, err := githubClient.ListRunners(ctx, owner, repo, nil)
				if err != nil {
					return fmt.Errorf("failed to create github client; %w", err)
				}
				for _, r := range runners {
					if r.Online {
						continue
					}
					err := githubClient.RemoveRunner(ctx, owner, repo, r.ID)
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
