package github

import (
	"context"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v33/github"
)

type Registerer interface {
	CreateRegistrationToken(context.Context, string, string) (string, error)
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

// CreateRegistrationToken creates Actions token to register self-hosted runner for repository
func (c *Client) CreateRegistrationToken(
	ctx context.Context,
	repositoryOwnerName string,
	repositoryName string,
) (string, error) {
	token, _, err := c.client.Actions.CreateRegistrationToken(
		ctx,
		repositoryOwnerName,
		repositoryName,
	)
	if err != nil {
		return "", err
	}

	return token.GetToken(), nil
}

type FakeClient struct{}

// NewFakeClient creates GitHub Actions Client
func NewFakeClient(
	appID int64,
	appInstallationID int64,
	appPrivateKeyPath string,
) (*FakeClient, error) {
	return &FakeClient{}, nil
}

// CreateRegistrationToken returns dummy token
func (c *FakeClient) CreateRegistrationToken(
	ctx context.Context,
	repositoryOwnerName string,
	repositoryName string,
) (string, error) {
	return "AAA", nil
}
