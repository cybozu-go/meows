[![GitHub release](https://img.shields.io/github/release/cybozu-go/meows.svg?maxAge=60)][releases]
[![CI](https://github.com/cybozu-go/meows/workflows/main/badge.svg)](https://github.com/cybozu-go/meows/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/cybozu-go/meows?tab=overview)](https://pkg.go.dev/github.com/cybozu-go/meows?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/meows)](https://goreportcard.com/report/github.com/cybozu-go/meows)

meows
=========================

meows is a Kubernetes controller for GitHub Actions [self-hosted runners](https://docs.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners).
It enables us to run GitHub Actions workflows on `Pod`s running on you Kubernetes
clusters.

**Project Status**: Initial development

Features
--------

- Self-hosted runner pool
  - Run a time-consuming startup script before a job starts
- Notification
  - Users can notice that a job succeeds or not with notification
- Workflow failure investigation
  - Users can extend the lifetime of a `Pod` and investigate what causes the failure

Documentation
-------------

[docs](docs/) directory contains documents about designs and specifications.

Docker images
-------------

Docker images are available on [Quay.io](https://quay.io/repository/cybozu)
- [Controller](https://quay.io/repository/cybozu/meows-controller)
- [Runner](https://quay.io/repository/cybozu/meows-runner)

[releases]: https://github.com/cybozu-go/meows/releases
