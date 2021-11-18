# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

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

[Unreleased]: https://github.com/cybozu-go/meows/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/cybozu-go/meows/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/cybozu-go/meows/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/cybozu-go/meows/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/cybozu-go/meows/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cybozu-go/meows/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cybozu-go/meows/compare/0a217cb1de9225c7eba5469ae8b286548a854333...v0.1.0
