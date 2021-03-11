/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var config struct {
	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string

	tokenSweepInterval time.Duration
	appID              int64
	appInstallationID  int64
	appPrivateKeyPath  string

	organizationName string
	repositoryName   string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "github-actions-controller",
	Short: "Kubernetes controller for GitHub Actions self-hosted runner",
	Long:  `Kubernetes controller for GitHub Actions self-hosted runner`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(config.organizationName) == 0 {
			return errors.New("organization-name should be specified")
		}
		if len(config.repositoryName) == 0 {
			return errors.New("repository-name should be specified")
		}
		if config.appID == 0 {
			return errors.New("app-id should be specified")
		}
		if config.appInstallationID == 0 {
			return errors.New("app-id should be specified")
		}
		if len(config.appPrivateKeyPath) == 0 {
			return errors.New("app-private-key-path should be specified")
		}
		return run()
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
	fs.StringVar(&config.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	fs.BoolVar(&config.enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	fs.StringVar(&config.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	fs.StringVarP(&config.organizationName, "organization-name", "o", "", "The GitHub organization name")
	fs.StringVarP(&config.repositoryName, "repository-name", "r", "", "The GitHub repository name")

	fs.DurationVar(&config.tokenSweepInterval, "token-fetch-interval", 30*time.Minute, "Interval to fetch GitHub Actions tokens.")
	fs.Int64Var(&config.appID, "app-id", 0, "The ID for GitHub App")
	fs.Int64Var(&config.appInstallationID, "app-installation-id", 0, "The installation ID for GitHub App")
	fs.StringVar(&config.appPrivateKeyPath, "app-private-key-path", "", "The path for GitHub App private key")
}
