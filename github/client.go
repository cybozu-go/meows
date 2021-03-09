package github

import (
	"context"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v33/github"
)

type Registerer interface {
	CreateOrganizationRegistrationToken(context.Context, string) (string, error)
}

type Client struct {
	client *github.Client
}

// NewClient creates GitHub Actions Client
func NewClient(
	appID int64,
	appInstallationID int64,
	appPrivateKeyPath string,
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
	return &Client{github.NewClient(&http.Client{Transport: rt})}, nil
}

// CreateOrganizationRegistrationToken creates Actions token to register self-hosted runner for repository
func (c *Client) CreateOrganizationRegistrationToken(
	ctx context.Context,
	organizationName string,
) (string, error) {
	token, _, err := c.client.Actions.CreateOrganizationRegistrationToken(
		ctx,
		organizationName,
	)
	if err != nil {
		return "", err
	}

	return token.GetToken(), nil
}

// FakeClient is fake client for GitHub Actions
type FakeClient struct{}

// NewFakeClient creates GitHub Actions Client
func NewFakeClient(
	appID int64,
	appInstallationID int64,
	appPrivateKeyPath string,
) (*FakeClient, error) {
	return &FakeClient{}, nil
}

// CreateOrganizationRegistrationToken returns dummy token
func (c *FakeClient) CreateOrganizationRegistrationToken(
	ctx context.Context,
	organizationName string,
) (string, error) {
	return "AAA", nil
}
