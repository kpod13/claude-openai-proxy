## Why

Builds and releases are currently done manually, which slows down the release process and introduces risk of human error. Automating via GitHub Actions will provide reproducible cross-platform builds and publish binaries to GitHub Releases on every version tag.

## What Changes

- New GitHub Actions workflow `.github/workflows/release.yml` triggered on `v*.*.*` tag pushes
- Workflow builds binaries for all target platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64, freebsd/amd64, freebsd/arm64
- Raw binaries are published as a GitHub Release with auto-generated release notes from git history

## Capabilities

### New Capabilities

- `release`: Automated cross-platform build and GitHub Release publication triggered by a version tag push

### Modified Capabilities

- `ci`: Release workflow gates on lint and test passing before publishing artifacts

## Impact

- New file: `.github/workflows/release.yml`
- Uses native `go build` with platform matrix and `gh release create`
- Requires `GITHUB_TOKEN` with `contents: write` (provided automatically by GitHub Actions)
- Affects: CI/CD pipeline, release process