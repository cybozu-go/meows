package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const postResultPath = "/result"

type postResultPayload struct {
	JobName      string `json:"job_name"`
	PodNamespace string `json:"pod_namespace"`
	PodName      string `json:"pod_name"`
	IsFailed     bool   `json:"is_failed"`
}

// NotifierClient is a client for Slack agent notifier.
type NotifierClient struct {
	notifierURL *url.URL
}

// NewNotifierClient creates NotifierClient.
func NewNotifierClient(addr string) (*NotifierClient, error) {
	u, err := url.Parse("http://" + addr)
	if err != nil {
		return nil, err
	}
	return &NotifierClient{u}, nil
}

// PostResult sends a POST request to notifier.
func (c *NotifierClient) PostResult(
	jobName string,
	podNamespace string,
	podName string,
	isFailed bool,
) error {
	p := postResultPayload{
		JobName:      jobName,
		PodNamespace: podNamespace,
		PodName:      podName,
		IsFailed:     isFailed,
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}

	u, err := c.notifierURL.Parse(postResultPath)
	if err != nil {
		return err
	}
	res, err := http.Post(
		u.String(),
		"application/json",
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"status code should be %d, but got %d",
			http.StatusOK,
			res.StatusCode,
		)
	}
	return nil
}
