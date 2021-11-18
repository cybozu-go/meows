package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/cybozu-go/meows/runner"
)

// These colors are based on the following guide.
// - Model Color Palette for Color Universal Design (ver.4)
//   ref: https://jfly.uni-koeln.de/colorset/
const (
	colorGreen  = "#03af7a" // RGB(3,175,122)
	colorRed    = "#ff4b00" // RGB(255,75,0)
	colorYellow = "#fff100" // RGB(255,241,0)
	colorGray   = "#84919e" // RGB(132,145,158)
)

var colors = map[string]string{
	runner.JobResultSuccess:   colorGreen,
	runner.JobResultFailure:   colorRed,
	runner.JobResultCancelled: colorGray,
	runner.JobResultUnknown:   colorYellow,
}

var captions = map[string]string{
	runner.JobResultSuccess:   "Success",
	runner.JobResultFailure:   "Failure",
	runner.JobResultCancelled: "Cancelled",
	runner.JobResultUnknown:   "Finished(Unknown)",
}

func makePayload(result string, namespaceName, podName string, info *runner.JobInfo) *resultAPIPayload {
	color, ok := colors[result]
	if !ok {
		color = colors[runner.JobResultUnknown]
	}
	head, ok := captions[result]
	if !ok {
		head = captions[runner.JobResultUnknown]
	}

	var text, job, pod string
	if info != nil {
		text = fmt.Sprintf("%s: %s's CI job in <%s|%s>", head, info.Actor, info.RepositoryURL(), info.Repository)
		job = fmt.Sprintf("<%s|%s #%d> [%s]", info.WorkflowURL(), info.WorkflowName, info.RunNumber, info.JobID)
		pod = fmt.Sprintf("%s/%s", namespaceName, podName)
	} else {
		text = fmt.Sprintf("%s: (failed to get job status)", head)
		job = "(unknown)"
		pod = fmt.Sprintf("%s/%s", namespaceName, podName)
	}

	return &resultAPIPayload{
		Color: color,
		Text:  text,
		Job:   job,
		Pod:   pod,
	}
}

// Client is a client for Slack agent.
type Client struct {
	serverURL *url.URL
	client    *http.Client
}

// NewClient creates Client.
func NewClient(serverURL string) (*Client, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// The communication with slack-agent does not use Proxy because it is an In-Cluster communication.
	transport.Proxy = nil
	return &Client{
		serverURL: u,
		client: &http.Client{
			Transport: transport,
		},
	}, nil
}

func (c *Client) UpdateServerURL(serverURL string) error {
	u, err := url.Parse(serverURL)
	if err != nil {
		return err
	}
	c.serverURL = u
	return nil
}

// PostResult sends a result of CI job to server.
func (c *Client) PostResult(ctx context.Context, channel, result string, extend bool, namespaceName, podName string, info *runner.JobInfo) error {
	payload := makePayload(result, namespaceName, podName, info)
	payload.Channel = channel
	payload.Extend = extend

	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	u, err := c.serverURL.Parse(resultAPIPath)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("status code: %d; failed to read response body: %s", res.StatusCode, err)
		}
		return fmt.Errorf("status code: %d; %s", res.StatusCode, string(data))
	}
	return nil
}
