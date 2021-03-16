package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v33/github"
)

// RegistrationTokenGenerator generates token for GitHub Action selfhosted runner
type RegistrationTokenGenerator interface {
	GetOrganizationName() string
	CreateRegistrationToken(context.Context, string) (string, error)
	ListRunners(context.Context, string) ([]*github.Runner, error)
	RemoveRunner(context.Context, string, int64) error
}

// Client is GitHub Client wrapper
type Client struct {
	client           *github.Client
	organizationName string
}

// NewClient creates GitHub Actions Client
func NewClient(
	appID int64,
	appInstallationID int64,
	appPrivateKeyPath string,
	organizationName string,
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
	}, nil
}

// GetOrganizationName returns organizationName.
func (c *Client) GetOrganizationName() string {
	return c.organizationName
}

// CreateRegistrationToken creates an Actions token to register self-hosted runner to the organization.
func (c *Client) CreateRegistrationToken(ctx context.Context, repositoryName string) (string, error) {
	token, _, err := c.client.Actions.CreateRegistrationToken(
		ctx,
		c.organizationName,
		repositoryName,
	)
	if err != nil {
		return "", err
	}

	return token.GetToken(), nil
}

// ListRunners lists registered self-hosted runners for the organization.
func (c *Client) ListRunners(ctx context.Context, repositoryName string) ([]*github.Runner, error) {
	var runners []*github.Runner

	opts := github.ListOptions{PerPage: 100}
	for {
		list, res, err := c.client.Actions.ListRunners(
			ctx,
			c.organizationName,
			repositoryName,
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

// RemoveRunner deletes an Actions runner of the organization.
func (c *Client) RemoveRunner(ctx context.Context, repositoryName string, runnerID int64) error {
	res, err := c.client.Actions.RemoveRunner(
		ctx,
		c.organizationName,
		repositoryName,
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

// FakeClient is a fake client
type FakeClient struct {
	organizationName string
	runners          map[string][]*github.Runner
}

// NewFakeClient creates GitHub Actions Client.
func NewFakeClient(organizationName string) *FakeClient {
	return &FakeClient{organizationName: organizationName}
}

// GetOrganizationName returns organizationName.
func (c *FakeClient) GetOrganizationName() string {
	return c.organizationName
}

// CreateRegistrationToken returns dummy token.
func (c *FakeClient) CreateRegistrationToken(ctx context.Context, repositoryName string) (string, error) {
	return "AAA", nil
}

// ListRunners returns dummy list.
func (c *FakeClient) ListRunners(ctx context.Context, repositoryName string) ([]*github.Runner, error) {
	return c.runners[repositoryName], nil
}

// RemoveRunner does not delete anything and returns success.
func (c *FakeClient) RemoveRunner(ctx context.Context, repositoryName string, runnerID int64) error {
	// skip existance and nil check below because this is mock
	runners := c.runners[repositoryName]
	for i, v := range runners {
		if *v.ID == runnerID {
			c.runners[repositoryName] = append(runners[:i], runners[i+1:]...)
			return nil
		}
	}
	return errors.New("not exist")
}

// SetRunners sets runners for multiple repositories
func (c *FakeClient) SetRunners(runners map[string][]*github.Runner) {
	c.runners = runners
}
