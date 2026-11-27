# Release procedure

This document describes how to release a new version of meows.

## 1. Versioning

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

## 2. Prepare change log entries

Release notes are generated from the pull requests merged since the previous release.
Label them according to [.github/release.yaml](.github/release.yaml) so they are categorized correctly.

## 3. Bump version

The version of meows is managed only in the [VERSION](VERSION) file.
It is embedded into the binaries at build time, and the manifests under `config/` refer to the `latest` image, so no other files need to be updated.

> [!NOTE]
> Renovate's "Routine dependency updates" pull requests that update base images in Dockerfiles also bump the patch version in `VERSION`.
> Merging such a pull request releases a new version, so you do not need to follow the steps below for it.

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

    On pull requests that change `VERSION`, the [release workflow](.github/workflows/release.yaml) fails if `VERSION` is not in the `X.Y.Z` format or the tag `vX.Y.Z` already exists.

## 4. Release

After the change is merged into `main`, the [release workflow](.github/workflows/release.yaml) runs as follows:

1. Build and push the runner images (`ghcr.io/cybozu-go/meows-runner`) from the merged commit,
   using the GitHub Actions runner version in [runner-images/runner.json](runner-images/runner.json).
2. Build and push the controller image (`ghcr.io/cybozu-go/meows-controller`) tagged as `X.Y.Z` and `latest`.
3. Create the tag `vX.Y.Z` and publish the GitHub release with generated release notes.

The runner images are pushed with the following tags:

| Tag | Description |
| --- | --- |
| `<os>` | Latest image for the OS (e.g. `ubuntu24.04`) |
| `<os>-meows<X.Y.Z>` | Latest image for the OS and meows version |
| `<runner version>-<os>-meows<X.Y.Z>` | Immutable tag, pushed only if it does not exist yet |
| `latest` | Same as the newest OS image |

The workflow runs on every push to `main` that changes `VERSION` or `runner-images/runner.json`.
If the tag `vX.Y.Z` already exists, steps 2 and 3 are skipped, and only the runner images are rebuilt from the existing tag `vX.Y.Z`.
This is how runner images are updated for a new GitHub Actions runner version without releasing a new meows version (see [docs/maintenance.md](docs/maintenance.md)).

If the workflow fails before creating the Git tag, fix the issue and rerun the workflow.
If the Git tag was created but the GitHub Release failed, create the release manually:

```
gh release create "vX.Y.Z" \
  --verify-tag \
  -t "Release vX.Y.Z" \
  --generate-notes
```

Once a new meows version is released, it will be available at <https://github.com/cybozu-go/meows/releases>.

[semver]: https://semver.org/spec/v2.0.0.html
