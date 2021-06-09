package main

import (
	"time"

	"github.com/cybozu-go/github-actions-controller/cmd/entrypoint/cmd"
)

func main() {
	cmd.Execute()
	time.Sleep(time.Duration(1<<63 - 1))
}
