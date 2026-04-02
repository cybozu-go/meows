# Release procedure

This document describes how to release a new version of meows.

## 1. Versioning

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

## 2. Prepare change log entries

Before you release, label pull requests appropriately. The release workflow creates a release note referring to the labels on the pull requests.
You can check the rules for labeling in [release.yml](.github/release.yaml).

## 3. Bump version

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

3. Bump version.

    ```bash
    echo "$VERSION" > VERSION
    sed -i -E "s/(.*newTag: ).*/\1${VERSION}/" config/controller/kustomization.yaml config/agent/kustomization.yaml
    sed -i -E "s/(.*Version = ).*/\1\"${VERSION}\"/" constants.go
    ```

4. Commit the change and push it.

    ```bash
    git commit -a -m "Bump version to $VERSION"
    git push origin "bump-$VERSION"
    ```

5. Create a pull request and merge it after review.

After the change is merged into `main`, the release workflow will automatically build and push artifacts.
Once the automated workflow completes, the release will be available at <https://github.com/cybozu-go/meows/releases>.

[semver]: https://semver.org/spec/v2.0.0.html
