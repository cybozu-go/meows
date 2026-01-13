package github

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/go-github/v80/github"
)

type FakeClientFactory struct {
	mu                sync.Mutex
	runners           map[string][]*Runner
	expiredAtDuration time.Duration
}

func NewFakeClientFactory() *FakeClientFactory {
	return &FakeClientFactory{
		runners:           map[string][]*Runner{},
		expiredAtDuration: 1 * time.Hour,
	}
}

func genKey(owner, repo string) string {
	if repo == "" {
		return owner
	}
	return owner + "/" + repo
}

func (f *FakeClientFactory) New(_ *ClientCredential) (Client, error) {
	return &FakeClient{parent: f}, nil
}

func (f *FakeClientFactory) createRegistrationToken(ctx context.Context, owner, repo string) (*github.RegistrationToken, error) {
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
func (f *FakeClientFactory) ListRunners(ctx context.Context, owner, repo string, labels []string) ([]*Runner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := genKey(owner, repo)

	ret := []*Runner{}
	runners := f.runners[key]
	for _, r := range runners {
		if r.hasLabels(labels) {
			ret = append(ret, r)
		}
	}
	return ret, nil
}

// RemoveRunner does not delete anything and returns success.
func (f *FakeClientFactory) RemoveRunner(ctx context.Context, owner, repo string, runnerID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := genKey(owner, repo)

	// skip existence and nil check below because this is mock
	runners := f.runners[key]
	for i, v := range runners {
		if v.ID == runnerID {
			f.runners[key] = append(runners[:i], runners[i+1:]...)
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

// CreateRegistrationToken returns dummy token.
func (c *FakeClient) CreateRegistrationToken(ctx context.Context, owner, repo string) (*github.RegistrationToken, error) {
	return c.parent.createRegistrationToken(ctx, owner, repo)
}

// ListRunners returns dummy list.
func (c *FakeClient) ListRunners(ctx context.Context, owner, repo string, labels []string) ([]*Runner, error) {
	return c.parent.ListRunners(ctx, owner, repo, labels)
}

// RemoveRunner does not delete anything and returns success.
func (c *FakeClient) RemoveRunner(ctx context.Context, owner, repo string, runnerID int64) error {
	return c.parent.RemoveRunner(ctx, owner, repo, runnerID)
}
