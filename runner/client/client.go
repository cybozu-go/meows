package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
)

type DeletionTimePayload struct {
	DeletionTime time.Time `json:"deletion_time"`
}

type Client interface {
	GetDeletionTime(ctx context.Context, ip string) (time.Time, error)
	PutDeletionTime(ctx context.Context, ip string, tm time.Time) error
}

type clientImpl struct {
	client http.Client
}

func (c *clientImpl) GetDeletionTime(ctx context.Context, ip string) (time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getDeletionTimeURL(ip), nil)
	if err != nil {
		return time.Time{}, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return time.Time{}, err
	}

	if res.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("runner pod (%s) return %d", ip, res.StatusCode)
	}

	dt := DeletionTimePayload{}
	if err := json.Unmarshal(b, &dt); err != nil {
		return time.Time{}, err
	}

	return dt.DeletionTime, nil
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

func getDeletionTimeURL(ip string) string {
	return fmt.Sprintf("http://%s:%d/%s", ip, constants.RunnerListenPort, constants.DeletionTimeEndpoint)
}

func NewClient() Client {
	return &clientImpl{}
}
