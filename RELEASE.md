# Release procedure

This document describes how to release a new version of meows.

## Versioning

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

## Prepare change log entries

Before you release, label pull requests appropriately. The release workflow creates a release note referring to the labels on the pull requests.
You can check the rules for labeling in [release.yml](.github/release.yaml).

## Run release workflow

1. Go to the [release meows workflow on GitHub Actions](https://github.com/cybozu-go/meows/actions/workflows/release-meows.yaml).
1. Click the "Run workflow" button.
1. Enter the new version `X.Y.Z` in the input box.
   - Without `v` prefix.
   - e.g., `0.21.6`
1. Click the green "Run workflow" button to start the workflow.

GitHub actions will build and push artifacts such as container images and
create a new GitHub release.
