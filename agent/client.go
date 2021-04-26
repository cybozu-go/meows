package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const postResultPath = "/result"

// NotifierClient is a client for Slack agent notifier.
type NotifierClient struct {
	// this could be *http.Client if needed, but keep it simple for now.
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
	repositoryName string,
	organizationName string,
	workflowName string,
	branchName string,
	runID uint,
	podNamespace string,
	podName string,
	isFailed bool,
) error {
	p := newPostResultPayload(
		repositoryName,
		organizationName,
		workflowName,
		branchName,
		runID,
		podNamespace,
		podName,
		isFailed,
	)

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
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("status code: %d, failed to read response body: %s", res.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, error: %s", res.StatusCode, string(data))
	}

	return nil
}
