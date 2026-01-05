# Release procedure

This document describes how to release a new version of meows.

## 1. Versioning

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

## 2. Bump version

1. Determine a new version number. Then set `VERSION` variable.

    ```bash
    # Set VERSION and confirm it. It should not have "v" prefix.
    VERSION=x.y.z
    echo $VERSION
    ```

2. Make a branch to release.

    ```bash
    git switch -c "bump-$VERSION"
    ```

3. Bump image version.

    ```bash
    sed -i -E "s/(.*newTag: ).*/\1${VERSION}/" config/controller/kustomization.yaml config/agent/kustomization.yaml
    sed -i -E "s/(.*Version = ).*/\1\"${VERSION}\"/" constants.go
    ```

4. Commit the change and push it.

    ```bash
    git commit -a -m "Bump version to $VERSION"
    git push origin "bump-$VERSION"
    ```

5. Create a pull request and merge it after review.

## 3. Prepare change log entries

Before you release, label pull requests appropriately. The release workflow creates a release note referring to the labels on the pull requests.
You can check the rules for labeling in [release.yml](.github/release.yaml).

## 4. Run release workflow

1. Go to the [release meows workflow on GitHub Actions](https://github.com/cybozu-go/meows/actions/workflows/release-meows.yaml).
1. Click the "Run workflow" button.
1. Enter the new version `X.Y.Z` in the input box.
   - Without `v` prefix.
   - e.g., `0.21.6`
1. Click the green "Run workflow" button to start the workflow.

GitHub actions will build and push artifacts such as container images and
create a new GitHub release.

[semver]: https://semver.org/spec/v2.0.0.html
