[![GitHub release](https://img.shields.io/github/release/cybozu-go/meows.svg?maxAge=60)][releases]
[![CI](https://github.com/cybozu-go/meows/workflows/main/badge.svg)](https://github.com/cybozu-go/meows/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/cybozu-go/meows?tab=overview)](https://pkg.go.dev/github.com/cybozu-go/meows?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/meows)](https://goreportcard.com/report/github.com/cybozu-go/meows)

# meows

`meows` is a Kubernetes controller for GitHub Actions [self-hosted runners](https://docs.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners).
You can run jobs in your GitHub Actions workflows on your Kubernetes cluster with meows.

**Project Status**: Initial development

## Supported software

- Kubernetes: 1.20, 1.21

## Features

- Run a self-hosted runner on a pod
  - We call the pod `runner pod` :)
  - You can customize the runner pod spec. E.g., labels, annotations, environment variables, volumes, and so on.
- Run GitHub Actions jobs in the clean environment
  - meows only runs one job on a single runner pod.
    When a job is done, meows will delete the runner pod to which the job is assigned and create the new one.
    So you can always run a job on a clean runner pod.
- Pool and control some self-hosted runners
  - meows prepares multiple runner pods as you specified.
- Extend the lifetime of runner pods
  - When a job is done, the assigned runner pod will be deleted after a while.
    But if necessary, you can extend the lifetime of the runner pod.
    For example, it enables you to do a failed investigation of a job.
  - Now, you can only extend if a job is failed.

## Documentation

[docs](docs/) directory contains documents about designs and specifications.

## Docker images

Docker images are available on [Quay.io](https://quay.io/repository/cybozu)
- [Controller](https://quay.io/repository/cybozu/meows-controller)
- [Runner](https://quay.io/repository/cybozu/meows-runner)

[releases]: https://github.com/cybozu-go/meows/releases
