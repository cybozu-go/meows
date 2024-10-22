# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.16.0] - 2024-09-26

- Support Kubernetes 1.30 ([#185](https://github.com/cybozu-go/meows/pull/185))

## [0.15.0] - 2024-06-20

### Changed

- Support Kubernetes 1.29 ([#183](https://github.com/cybozu-go/meows/pull/183))
- Build with go 1.22 ([#183](https://github.com/cybozu-go/meows/pull/183))

## [0.14.0] - 2024-02-14

### Breaking Changes

#### Migrate image registry

We migrated the image repositories of meows to `ghcr.io`.
From meows v0.14.0, please use the following images.

- <https://github.com/cybozu-go/meows/pkgs/container/meows-controller>
- <https://github.com/cybozu-go/meows/pkgs/container/meows-runner>

The images on Quay.io ([meows-controller](https://quay.io/repository/cybozu/meows-controller), [meows-runner](https://quay.io/repository/cybozu/meows-runner)) will not be updated in the future.

### Changed

- Migrate to ghcr.io (#180)

## [0.13.0] - 2023-10-31

### Changed

- Support Kubernetes 1.27 ([#178](https://github.com/cybozu-go/meows/pull/1781))
- Build with go 1.21 ([#178](https://github.com/cybozu-go/meows/pull/178))

## [0.12.0] - 2023-07-05

### Changed

- Improve runner extend UI on Slack message ([#175](https://github.com/cybozu-go/meows/pull/175))

### Fixed

- Fix Slack notification message layout ([#173](https://github.com/cybozu-go/meows/pull/173))

## [0.11.0] - 2023-05-12

### Changed

- Support Kubernetes 1.26 ([#171](https://github.com/cybozu-go/meows/pull/171))
- Build with go 1.20 ([#171](https://github.com/cybozu-go/meows/pull/171))
- Update runner version to 2.304.0 ([#169](https://github.com/cybozu-go/meows/pull/169))

### Fixed

- Update workflow for test repository ([#170](https://github.com/cybozu-go/meows/pull/170))

## [0.10.0] - 2023-02-07

### Changed

- Support Kubernetes 1.25 ([#166](https://github.com/cybozu-go/meows/pull/166))
- Build with go 1.19 ([#167](https://github.com/cybozu-go/meows/pull/167))
- Update Ubuntu base image ([#167](https://github.com/cybozu-go/meows/pull/167))
- Update runner version to 2.301.1 ([#167](https://github.com/cybozu-go/meows/pull/167))

## [0.9.1] - 2022-12-16

### Changed

- Update runner version to 2.300.0 ([#163](https://github.com/cybozu-go/meows/pull/163))

## [0.9.0] - 2022-10-17

### Added

- Add denyDisruption field to RunnerPool (#153)

### Fixed

- Fix a race condition around metrics handling (#158)

## [0.8.0] - 2022-07-25

### Added

- Add envFrom field to RunnerPool (#150)

### Changed

- Support Kubernetes 1.23 and 1.24 (#151)
- Build with go 1.18 (#151)
- Update dependencies (#151)

## [0.7.0] - 2022-04-13

### Added

- Add nodeSelector and tolerations field (#145)

### Changed

- Update runner version to 2.289.2 (#147)

### Fixed

- Ignore not running pods when maintaining runner pods. (#143)

## [0.6.2] - 2022-03-23

### Changed

- Update runner version to 2.288.1 (#138)

## [0.6.1] - 2022-01-25

### Fixed

- Fixed a bug that the meows-controller does not delete finished runner pods immediately. (#134)

## [0.6.0] - 2022-01-11

### Changed

- Refine RunnerPool CRD (#130)
- Read GitHub credential from RunnerPool's namespace (#116)

### Fixed

- Fix the slack notifications failure when updating a RunnerPool (#125)

## [0.5.0] - 2021-12-06

### Added

- Support Kubernetes 1.22 (#123)

### Changed

- Remove --repository-names option from meows-controller (#113)
- Stop creating the latest tag images (#114)
- Build with go 1.17 (#122)

## [0.4.2] - 2021-11-18

### Fixed

- Reuse the slack-agent client (#107)

## [0.4.1] - 2021-11-12

### Fixed

- Fix the way to mount a secret that does not use subPath. (#103)

## [0.4.0] - 2021-11-01

### Added

- Support k8s 1.21 (#76)
- Add WorkVolume field to RunnerPodTemplateSec (#82)
- Add MaxRunnerPods field to RunnerPoolSpec (#83, #98)
- Pass runner token via secret resource (#45, #88)
- Recreate unused runner pods (#90)
- Add status API for runner pods (#89)
- Support organization-level runner (#97)

### Changed

- Change LICENSE from MIT to Apache 2 (#70)
- Change service account for the meows controller (#80)
- Unlink busy runner (#79)
- Use emptyDir for work dir (#81)
- Change once option to ephemeral option (#94)

### Fixed

- Fix runner pod extension (#78)

## [0.3.1] - 2021-08-13

### Fixed

- Update latest runner image (#67)

## [0.3.0] - 2021-08-13

### Added

- Add meows command (#62)

### Changed

- Set name prefix for controller resources (#63)

## [0.2.0] - 2021-08-11

### Changed

- Everything.

## [0.1.0] - 2021-04-13

### Added

- Implement github-actions-controller at minimal (#1)

[Unreleased]: https://github.com/cybozu-go/meows/compare/v0.16.0...HEAD
[0.16.0]: https://github.com/cybozu-go/meows/compare/v0.15.0...v0.16.0
[0.15.0]: https://github.com/cybozu-go/meows/compare/v0.14.0...v0.15.0
[0.14.0]: https://github.com/cybozu-go/meows/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/cybozu-go/meows/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/cybozu-go/meows/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/cybozu-go/meows/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/cybozu-go/meows/compare/v0.9.1...v0.10.0
[0.9.1]: https://github.com/cybozu-go/meows/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/cybozu-go/meows/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/cybozu-go/meows/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/cybozu-go/meows/compare/v0.6.2...v0.7.0
[0.6.2]: https://github.com/cybozu-go/meows/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/cybozu-go/meows/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/cybozu-go/meows/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/cybozu-go/meows/compare/v0.4.2...v0.5.0
[0.4.2]: https://github.com/cybozu-go/meows/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/cybozu-go/meows/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/cybozu-go/meows/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/cybozu-go/meows/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/cybozu-go/meows/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cybozu-go/meows/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cybozu-go/meows/compare/0a217cb1de9225c7eba5469ae8b286548a854333...v0.1.0
