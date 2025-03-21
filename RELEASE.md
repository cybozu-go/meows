# Release procedure

This document describes how to release a new version of meows.

## Versioning

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

## Prepare change log entries

Add notable changes since the last release to [CHANGELOG.md](CHANGELOG.md).
It should look like:

```markdown
(snip)
## [Unreleased]

### Added

- Implement ... (#35)

### Changed

- Fix a bug in ... (#33)

### Removed

- Deprecated `-option` is removed ... (#39)

(snip)
```

## Bump version

1. Determine a new version number. Then set `VERSION` variable.

    ```bash
    # Set VERSION and confirm it. It should not have "v" prefix.
    VERSION=x.y.z
    echo $VERSION
    ```

2. Make a branch to release

    ```bash
    git switch -c "bump-$VERSION"
    ```

3. Edit `CHANGELOG.md` for the new version ([example][]).
4. Bump image version.

    ```bash
    sed -i -E "s/(.*newTag: ).*/\1${VERSION}/" config/controller/kustomization.yaml config/agent/kustomization.yaml
    sed -i -E "s/(.*Version = ).*/\1\"${VERSION}\"/" constants.go
    ```

5. Commit the change and push it.

    ```bash
    git commit -a -m "Bump version to $VERSION"
    git push origin "bump-$VERSION"
    ```

6. Merge this branch.
7. Add a git tag to the main HEAD, then push it.

    ```bash
    # Set VERSION again.
    VERSION=x.y.z
    echo $VERSION

    git checkout main
    git pull
    git tag -a -m "Release v$VERSION" "v$VERSION"

    # Make sure the release tag exists.
    git tag -ln | grep $VERSION

    git push origin "v$VERSION"
    ```

GitHub actions will build and push artifacts such as container images and
create a new GitHub release.

[semver]: https://semver.org/spec/v2.0.0.html
[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
