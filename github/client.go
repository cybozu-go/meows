package github

import (
	"context"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v33/github"
)

// RegistrationTokenGenerator generates token for GitHub Action selfhosted runner
type RegistrationTokenGenerator interface {
	CreateOrganizationRegistrationToken(context.Context) (string, error)
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

// CreateOrganizationRegistrationToken creates Actions token to register self-hosted runner for repository
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

type fakeClient struct{}

// NewfakeClient creates GitHub Actions Client
func NewFakeClient() *fakeClient {
	return &fakeClient{}
}

// CreateOrganizationRegistrationToken returns dummy token
func (c *fakeClient) CreateOrganizationRegistrationToken(ctx context.Context) (string, error) {
	return "AAA", nil
}
