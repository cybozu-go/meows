package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/agent"
)

type DeletionTimePayload struct {
	DeletionTime time.Time `json:"deletion_time"`
}

type JobResultResponse struct {
	Status       string         `json:"status"`
	Extend       bool           `json:"extend"`
	SlackChannel string         `json:"slack_channel"`
	PodNamespace string         `json:"pod_namespace"`
	PodName      string         `json:"pod_name"`
	JobInfo      *agent.JobInfo `json:"jobinfo"`
}

type Client interface {
	GetDeletionTime(ctx context.Context, ip string) (time.Time, error)
	PutDeletionTime(ctx context.Context, ip string, tm time.Time) error
	GetJobResult(ctx context.Context, ip string) (string, error)
}

type clientImpl struct {
	client http.Client
}

func NewClient() Client {
	return &clientImpl{}
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

func (c *clientImpl) GetJobResult(ctx context.Context, ip string) (*JobResultResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getJobResultURL(ip), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("runner pod (%s) return %d", ip, res.StatusCode)
	}

	jr := JobResultResponse{}

	if err := json.Unmarshal(b, &jr); err != nil {
		return nil, err
	}

	return &jr, nil
}

func getJobResultURL(ip string) string {
	return fmt.Sprintf("http://%s:%d/%s", ip, constants.RunnerListenPort, constants.RunnerJobResultEndPoint)
}

func getDeletionTimeURL(ip string) string {
	return fmt.Sprintf("http://%s:%d/%s", ip, constants.RunnerListenPort, constants.DeletionTimeEndpoint)
}

// FakeClient is a fake client
type FakeClient struct {
	deletionTimes map[string]time.Time
	jobResults    map[string]*JobResultResponse
}

func NewFakeClient() *FakeClient {
	return &FakeClient{
		deletionTimes: map[string]time.Time{},
		jobResults:    map[string]*JobResultResponse{},
	}
}

func (c *FakeClient) GetDeletionTime(ctx context.Context, ip string) (time.Time, error) {
	return c.deletionTimes[ip], nil
}

func (c *FakeClient) PutDeletionTime(ctx context.Context, ip string, tm time.Time) error {
	c.deletionTimes[ip] = tm
	return nil
}

func (c *FakeClient) SetDeletionTimes(ip string, tm time.Time) {
	c.deletionTimes[ip] = tm
}

func (c *FakeClient) GetJobResult(ctx context.Context, ip string) (*JobResultResponse, error) {
	return c.jobResults[ip], nil
}

func (c *FakeClient) SetJobResult(ip string, jr *JobResultResponse) {
	c.jobResults[ip] = jr
}
