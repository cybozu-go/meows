[![GitHub release](https://img.shields.io/github/release/cybozu-go/meows.svg?maxAge=60)][releases]
[![CI](https://github.com/cybozu-go/meows/workflows/main/badge.svg)](https://github.com/cybozu-go/meows/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/cybozu-go/meows?tab=overview)](https://pkg.go.dev/github.com/cybozu-go/meows?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/meows)](https://goreportcard.com/report/github.com/cybozu-go/meows)

# meows

`meows` is a Kubernetes controller for GitHub Actions [self-hosted runners](https://docs.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners).
You can run jobs in your GitHub Actions workflows on your Kubernetes cluster with meows.

**Project Status**: Initial development

## Supported software

- Kubernetes: 1.32, 1.33, 1.34

## Features

- Run a self-hosted runner on a pod
  - meows runs the [GitHub Actions Runner](https://github.com/actions/runner) on a pod.
    It allows you to use a pod as a self-hosted runner.
  - The pod spec is customizable. E.g., labels, annotations, environment variables, volumes, and other specs are customizable.
    So you can prepare any environment you want.
  - We call the pod `runner pod` :)
- Pool and maintain some runners and runner pods
  - meows prepares multiple runner pods as you specified.
    It allows you to pool various runners and reduce Actions job clagging.
- Run GitHub Actions jobs in the clean environment
  - meows only runs one job on a single runner pod.
    When a job in a runner pod gets finished, meows will delete the pod and create a new one.
    So you can always run a job on a clean runner pod.
- Extend the lifetimes of runner pods
  - When a job has finished, meows will delete the assigned runner pod after a while.
    But if necessary, you can extend the lifetime of the runner pod.
    For example, it enables you to investigate a failed job.
  - Currently, you can only extend it if a job has failed.

## Documentation

[docs](docs/) directory contains documents about designs and specifications.

## Docker images

Docker images are available on [ghcr.io](https://github.com/orgs/cybozu-go/packages?repo_name=meows)

- [Controller](https://github.com/cybozu-go/meows/pkgs/container/meows-controller)
- [Runner](https://github.com/cybozu-go/meows/pkgs/container/meows-runner)

[releases]: https://github.com/cybozu-go/meows/releases
