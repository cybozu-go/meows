package github

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/go-github/v41/github"
)

type FakeClientFactory struct {
	mu                sync.Mutex
	organizationName  string
	runners           map[string][]*Runner
	expiredAtDuration time.Duration
}

func NewFakeClientFactory(organizationName string) *FakeClientFactory {
	return &FakeClientFactory{
		organizationName:  organizationName,
		runners:           map[string][]*Runner{},
		expiredAtDuration: 1 * time.Hour,
	}
}

func (f *FakeClientFactory) New(_ context.Context, _ *ClientCredential) (Client, error) {
	return &FakeClient{parent: f}, nil
}

func (f *FakeClientFactory) getOrganizationName() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.organizationName
}

func (f *FakeClientFactory) createRegistrationToken(ctx context.Context, repositoryName string) (*github.RegistrationToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	fakeToken := "faketoken"
	return &github.RegistrationToken{
		Token: &fakeToken,
		ExpiresAt: &github.Timestamp{
			Time: time.Now().Add(f.expiredAtDuration),
		},
	}, nil
}

// ListRunners returns dummy list.
func (f *FakeClientFactory) ListRunners(ctx context.Context, repositoryName string, labels []string) ([]*Runner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ret := []*Runner{}
	runners := f.runners[repositoryName]
	for _, r := range runners {
		if r.hasLabels(labels) {
			ret = append(ret, r)
		}
	}
	return ret, nil
}

// RemoveRunner does not delete anything and returns success.
func (f *FakeClientFactory) RemoveRunner(ctx context.Context, repositoryName string, runnerID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// skip existence and nil check below because this is mock
	runners := f.runners[repositoryName]
	for i, v := range runners {
		if v.ID == runnerID {
			f.runners[repositoryName] = append(runners[:i], runners[i+1:]...)
			return nil
		}
	}
	return errors.New("not exist")
}

func (f *FakeClientFactory) SetRunners(runners map[string][]*Runner) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.runners = runners
}

func (f *FakeClientFactory) SetExpiredAtDuration(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.expiredAtDuration = d
}

// FakeClient is a fake client
type FakeClient struct {
	parent *FakeClientFactory
}

// GetOrganizationName returns organizationName.
func (c *FakeClient) GetOrganizationName() string {
	return c.parent.getOrganizationName()
}

// CreateRegistrationToken returns dummy token.
func (c *FakeClient) CreateRegistrationToken(ctx context.Context, repositoryName string) (*github.RegistrationToken, error) {
	return c.parent.createRegistrationToken(ctx, repositoryName)
}

// ListRunners returns dummy list.
func (c *FakeClient) ListRunners(ctx context.Context, repositoryName string, labels []string) ([]*Runner, error) {
	return c.parent.ListRunners(ctx, repositoryName, labels)
}

// RemoveRunner does not delete anything and returns success.
func (c *FakeClient) RemoveRunner(ctx context.Context, repositoryName string, runnerID int64) error {
	return c.parent.RemoveRunner(ctx, repositoryName, runnerID)
}
