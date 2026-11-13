# Release procedure

This document describes how to release a new version of meows.

## 1. Versioning

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

## 2. Prepare change log entries

Release notes are generated from the pull requests merged since the previous release.
Label them according to [.github/release.yaml](.github/release.yaml) so they are categorized correctly.

## 3. Bump version

The version of meows is managed only in the [VERSION](VERSION) file.
The controller embeds it at build time, and the manifests under `config/` refer to the `latest` image, so no other files need to be updated.

1. Determine a new version number.

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
    ```

4. Commit the change and push it.

    ```bash
    git commit -a -m "Bump version to $VERSION"
    git push origin "bump-$VERSION"
    ```

5. Create a pull request and merge it after review.

    The [release workflow](.github/workflows/release-meows.yaml) validates `VERSION` on the pull request. Make sure it passes before merging.

## 4. Release

After the change is merged into `main`, the release workflow automatically pushes the controller and runner images, creates the tag `vX.Y.Z`, and publishes the GitHub release.
The controller image is also tagged as `latest`.

Once the automated workflow completes, the release will be available at <https://github.com/cybozu-go/meows/releases>.

[semver]: https://semver.org/spec/v2.0.0.html
