package runner

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
)

type Client interface {
	GetDeletionTime(ip string) (string, error)
}

type ClientImpl struct{}

func (c *ClientImpl) GetDeletionTime(ip string) (string, error) {
	url := fmt.Sprintf("http://%s:%d/%s", ip, constants.RunnerMetricsPort, constants.DeletionTimeEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(b), "\n"), nil
}

func NewClient() Client {
	return &ClientImpl{}
}
