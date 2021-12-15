package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

const statusOnline = "online"

type Runner struct {
	ID     int64
	Name   string
	Online bool
	Busy   bool
	Labels []string
}

func convert(ghRunner *github.Runner) *Runner {
	var labels []string
	for _, l := range ghRunner.Labels {
		labels = append(labels, l.GetName())
	}
	return &Runner{
		Name:   ghRunner.GetName(),
		ID:     ghRunner.GetID(),
		Online: ghRunner.GetStatus() == statusOnline,
		Busy:   ghRunner.GetBusy(),
		Labels: labels,
	}
}

func (r *Runner) hasLabels(labels []string) bool {
	actualLabelMap := map[string]struct{}{}
	for _, l := range r.Labels {
		actualLabelMap[l] = struct{}{}
	}
	for _, required := range labels {
		if _, ok := actualLabelMap[required]; !ok {
			return false
		}
	}
	return true
}

// Client generates token for GitHub Action selfhosted runner
type Client interface {
	GetOrganizationName() string
	CreateRegistrationToken(context.Context, string) (*github.RegistrationToken, error)
	ListRunners(context.Context, string, []string) ([]*Runner, error)
	RemoveRunner(context.Context, string, int64) error
}

type ClientCredential struct {
	PersonalAccessToken string
	AppID               int64
	AppInstallationID   int64
	PrivateKey          []byte
	PrivateKeyPath      string
}

// ClientFactory is a factory of Clients.
type ClientFactory interface {
	New(*ClientCredential) (Client, error)
}

type defaultFactory struct {
	organizationName string
}

func NewFactory(organizationName string) ClientFactory {
	return &defaultFactory{
		organizationName: organizationName,
	}
}

func (f *defaultFactory) New(cred *ClientCredential) (Client, error) {
	switch {
	case len(cred.PersonalAccessToken) != 0:
		return newClientFromPAT(f.organizationName, cred.PersonalAccessToken), nil
	case len(cred.PrivateKey) != 0:
		return newClientFromAppKey(f.organizationName, cred.AppID, cred.AppInstallationID, cred.PrivateKey)
	case len(cred.PrivateKeyPath) != 0:
		return newClientFromAppKeyFile(f.organizationName, cred.AppID, cred.AppInstallationID, cred.PrivateKeyPath)
	default:
		return nil, errors.New("invalid credential")
	}
}

// clientWrapper is a wrapper of GitHub client.
type clientWrapper struct {
	client           *github.Client
	organizationName string
}

// newClientFromPAT creates GitHub Actions Client from a personal access token (PAT).
func newClientFromPAT(organizationName string, pat string) Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: pat},
	)
	tc := oauth2.NewClient(ctx, ts)
	return &clientWrapper{
		client:           github.NewClient(tc),
		organizationName: organizationName,
	}
}

// newClientFromAppKey creates GitHub Actions Client from a private key of a GitHub app.
func newClientFromAppKey(organizationName string, appID, appInstallationID int64, privateKey []byte) (Client, error) {
	rt, err := ghinstallation.New(http.DefaultTransport, appID, appInstallationID, privateKey)
	if err != nil {
		return nil, err
	}
	return &clientWrapper{
		client:           github.NewClient(&http.Client{Transport: rt}),
		organizationName: organizationName,
	}, nil
}

// newClientFromAPIKey creates GitHub Actions Client from a private key of a GitHub app.
func newClientFromAppKeyFile(organizationName string, appID, appInstallationID int64, privateKeyPath string) (Client, error) {
	rt, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, appInstallationID, privateKeyPath)
	if err != nil {
		return nil, err
	}
	return &clientWrapper{
		client:           github.NewClient(&http.Client{Transport: rt}),
		organizationName: organizationName,
	}, nil
}

// GetOrganizationName returns organizationName.
func (c *clientWrapper) GetOrganizationName() string {
	return c.organizationName
}

// CreateRegistrationToken creates an Actions token to register self-hosted runner to the organization.
func (c *clientWrapper) CreateRegistrationToken(ctx context.Context, repositoryName string) (*github.RegistrationToken, error) {
	var token *github.RegistrationToken
	var res *github.Response
	var err error
	if repositoryName == "" {
		token, res, err = c.client.Actions.CreateOrganizationRegistrationToken(
			ctx,
			c.organizationName,
		)
	} else {
		token, res, err = c.client.Actions.CreateRegistrationToken(
			ctx,
			c.organizationName,
			repositoryName,
		)
	}
	if e, ok := err.(*url.Error); ok {
		// When url.Error came back, it was because the raw Responce leaked out as a string.
		return nil, fmt.Errorf("failed to create registration token: %s %s", e.Op, e.URL)
	}
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("invalid status code %d", res.StatusCode)
	}

	return token, nil
}

// ListRunners lists registered self-hosted runners for the organization.
func (c *clientWrapper) ListRunners(ctx context.Context, repositoryName string, labels []string) ([]*Runner, error) {
	var runners []*Runner

	opts := github.ListOptions{PerPage: 100}
	for {
		var list *github.Runners
		var res *github.Response
		var err error
		if repositoryName == "" {
			list, res, err = c.client.Actions.ListOrganizationRunners(
				ctx,
				c.organizationName,
				&opts,
			)
		} else {
			list, res, err = c.client.Actions.ListRunners(
				ctx,
				c.organizationName,
				repositoryName,
				&opts,
			)
		}
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("invalid status code %d", res.StatusCode)
		}

		for _, ghRunner := range list.Runners {
			r := convert(ghRunner)
			if !r.hasLabels(labels) {
				continue
			}
			runners = append(runners, r)
		}
		if res.NextPage == 0 {
			break
		}

		opts.Page = res.NextPage
		time.Sleep(500 * time.Microsecond)
	}
	return runners, nil
}

// RemoveRunner deletes an Actions runner of the organization.
func (c *clientWrapper) RemoveRunner(ctx context.Context, repositoryName string, runnerID int64) error {
	var res *github.Response
	var err error
	if repositoryName == "" {
		res, err = c.client.Actions.RemoveOrganizationRunner(
			ctx,
			c.organizationName,
			runnerID,
		)
	} else {
		res, err = c.client.Actions.RemoveRunner(
			ctx,
			c.organizationName,
			repositoryName,
			runnerID,
		)
	}
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("invalid status code %d", res.StatusCode)
	}
	return nil
}
