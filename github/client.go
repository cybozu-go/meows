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
	CreateOrganizationRegistrationToken(context.Context) (string, error)
	ListOrganizationRunners(context.Context) ([]*github.Runner, error)
	RemoveOrganizationRunner(context.Context, int64) error
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
	return &Client{github.NewClient(&http.Client{Transport: rt}), organizationName}, nil
}

// CreateOrganizationRegistrationToken creates an Actions token to register self-hosted runner to the organization
func (c *Client) CreateOrganizationRegistrationToken(ctx context.Context) (string, error) {
	token, _, err := c.client.Actions.CreateOrganizationRegistrationToken(
		ctx,
		c.organizationName,
	)
	if err != nil {
		return "", err
	}

	return token.GetToken(), nil
}

// ListOrganizationRunners lists registered self-hosted runners for the organization
func (c *Client) ListOrganizationRunners(ctx context.Context) ([]*github.Runner, error) {
	var runners []*github.Runner

	opts := github.ListOptions{PerPage: 100}
	for {
		list, res, err := c.client.Actions.ListOrganizationRunners(
			ctx,
			c.organizationName,
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

// RemoveOrganizationRunner deletes an Actions runner of the organization
func (c *Client) RemoveOrganizationRunner(ctx context.Context, runnerID int64) error {
	res, err := c.client.Actions.RemoveOrganizationRunner(
		ctx,
		c.organizationName,
		runnerID,
	)
	if err != nil {
		return err
	}
	if res.StatusCode != 204 {
		return fmt.Errorf("status should be 204 but %d", res.StatusCode)

	}
	return nil
}

type fakeClient struct{}

// NewfakeClient creates GitHub Actions Client
func NewFakeClient() *fakeClient {
	return &fakeClient{}
}

// CreateOrganizationRegistrationToken returns dummy token
func (c *fakeClient) CreateOrganizationRegistrationToken(ctx context.Context) (string, error) {
	return "AAA", nil
}

// ListOrganizationRunners returns dummy list
func (c *fakeClient) ListOrganizationRunners(ctx context.Context) ([]*github.Runner, error) {
	return []*github.Runner{}, nil
}

// RemoveOrganizationRunner does not delete anything and returns success
func (c *fakeClient) RemoveOrganizationRunner(ctx context.Context, runnerID int64) error {
	return nil
}
