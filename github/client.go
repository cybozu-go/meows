package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v33/github"
)

// RegistrationTokenGenerator generates token for GitHub Action selfhosted runner
type RegistrationTokenGenerator interface {
	CreateRegistrationToken(context.Context) (string, error)
	ListRunners(context.Context) ([]*github.Runner, error)
	RemoveRunner(context.Context, int64) error
}

// Client is GitHub Client wrapper
type Client struct {
	client           *github.Client
	organizationName string
	repositoryName   string
}

// NewClient creates GitHub Actions Client
func NewClient(
	appID int64,
	appInstallationID int64,
	appPrivateKeyPath string,
	organizationName string,
	repositoryName string,
) (*Client, error) {
	rt, err := ghinstallation.NewKeyFromFile(
		http.DefaultTransport,
		appID,
		appInstallationID,
		appPrivateKeyPath,
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		client:           github.NewClient(&http.Client{Transport: rt}),
		organizationName: organizationName,
		repositoryName:   repositoryName,
	}, nil
}

// CreateRegistrationToken creates an Actions token to register self-hosted runner to the organization
func (c *Client) CreateRegistrationToken(ctx context.Context) (string, error) {
	token, _, err := c.client.Actions.CreateRegistrationToken(
		ctx,
		c.organizationName,
		c.repositoryName,
	)
	if err != nil {
		return "", err
	}

	return token.GetToken(), nil
}

// ListRunners lists registered self-hosted runners for the organization
func (c *Client) ListRunners(ctx context.Context) ([]*github.Runner, error) {
	var runners []*github.Runner

	opts := github.ListOptions{PerPage: 100}
	for {
		list, res, err := c.client.Actions.ListRunners(
			ctx,
			c.organizationName,
			c.repositoryName,
			&opts,
		)
		if err != nil {
			return nil, err
		}

		runners = append(runners, list.Runners...)
		if res.NextPage == 0 {
			break

		}
		opts.Page = res.NextPage
	}
	return runners, nil
}

// RemoveRunner deletes an Actions runner of the organization
func (c *Client) RemoveRunner(ctx context.Context, runnerID int64) error {
	res, err := c.client.Actions.RemoveRunner(
		ctx,
		c.organizationName,
		c.repositoryName,
		runnerID,
	)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("status should be %d but %d", http.StatusNoContent, res.StatusCode)

	}
	return nil
}

type fakeClient struct{}

// NewfakeClient creates GitHub Actions Client
func NewFakeClient() *fakeClient {
	return &fakeClient{}
}

// CreateRegistrationToken returns dummy token
func (c *fakeClient) CreateRegistrationToken(ctx context.Context) (string, error) {
	return "AAA", nil
}

// ListRunners returns dummy list
func (c *fakeClient) ListRunners(ctx context.Context) ([]*github.Runner, error) {
	return []*github.Runner{}, nil
}

// RemoveRunner does not delete anything and returns success
func (c *fakeClient) RemoveRunner(ctx context.Context, runnerID int64) error {
	return nil
}
