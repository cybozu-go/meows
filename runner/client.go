package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	constants "github.com/cybozu-go/meows"
)

type Client interface {
	PutDeletionTime(ctx context.Context, ip string, tm time.Time) error
	GetStatus(ctx context.Context, ip string) (*Status, error)
}

type clientImpl struct {
	client http.Client
}

func NewClient() Client {
	return &clientImpl{}
}

func (c *clientImpl) PutDeletionTime(ctx context.Context, ip string, tm time.Time) error {
	b, err := json.Marshal(DeletionTimePayload{
		DeletionTime: tm,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, getDeletionTimeURL(ip), bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("runner pod (%s) return %d, send data is %v", ip, res.StatusCode, string(b))
	}
	return nil
}

func (c *clientImpl) GetStatus(ctx context.Context, ip string) (*Status, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getStatusURL(ip), nil)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("runner pod (%s) return %d", ip, res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	s := Status{}
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func getStatusURL(ip string) string {
	return fmt.Sprintf("http://%s:%d/%s", ip, constants.RunnerListenPort, constants.StatusEndPoint)
}

func getDeletionTimeURL(ip string) string {
	return fmt.Sprintf("http://%s:%d/%s", ip, constants.RunnerListenPort, constants.DeletionTimeEndpoint)
}

// FakeClient is a fake client
type FakeClient struct {
	statuses map[string]*Status
}

func NewFakeClient() *FakeClient {
	return &FakeClient{
		statuses: map[string]*Status{},
	}
}

func (c *FakeClient) PutDeletionTime(ctx context.Context, ip string, tm time.Time) error {
	if _, ok := c.statuses[ip]; !ok {
		return fmt.Errorf("[FakeClient.PutDeletionTime] runner pod (%s) status is not defined", ip)
	}
	c.statuses[ip].DeletionTime = &tm
	return nil
}

func (c *FakeClient) GetStatus(ctx context.Context, ip string) (*Status, error) {
	if st, ok := c.statuses[ip]; ok {
		return st, nil
	}
	return nil, fmt.Errorf("[FakeClient.GetStatus] runner pod (%s) status is not defined", ip)
}

func (c *FakeClient) SetStatus(ip string, st *Status) {
	c.statuses[ip] = st
}
