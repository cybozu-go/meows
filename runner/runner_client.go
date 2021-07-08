package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	constants "github.com/cybozu-go/github-actions-controller"
)

type Client interface {
	GetDeletionTime(ctx context.Context, ip string) (DeletionTimePayload, error)
	PutDeletionTime(ctx context.Context, ip string, dt DeletionTimePayload) error
}

type ClientImpl struct {
	client http.Client
}

func (c *ClientImpl) GetDeletionTime(ctx context.Context, ip string) (DeletionTimePayload, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getDeletionTimeURL(ip), nil)
	if err != nil {
		return DeletionTimePayload{}, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return DeletionTimePayload{}, err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return DeletionTimePayload{}, err
	}

	if res.StatusCode != http.StatusOK {
		return DeletionTimePayload{}, fmt.Errorf("Runner Pod (%s) return not %d", ip, http.StatusOK)
	}

	dt := DeletionTimePayload{}
	if err := json.Unmarshal(b, &dt); err != nil {
		return DeletionTimePayload{}, err
	}

	return dt, nil
}

func (c *ClientImpl) PutDeletionTime(ctx context.Context, ip string, dt DeletionTimePayload) error {
	b, err := json.Marshal(dt)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, getDeletionTimeURL(ip), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Runner Pod (%s) return not %d, send data is %v", ip, http.StatusNoContent, dt)
	}
	return nil
}

func getDeletionTimeURL(ip string) string {
	return fmt.Sprintf("http://%s:%d/%s", ip, constants.RunnerListenPort, constants.DeletionTimeEndpoint)
}

func NewClient() Client {
	return &ClientImpl{}
}
