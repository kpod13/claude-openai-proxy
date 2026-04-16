## Context

The project is a Go server (`cmd/server/main.go`) built with standard `go build`. A CI workflow (lint + tests) already exists. Binaries are currently distributed manually; there is no automated GitHub Release process. The goal is a pipeline that triggers on a version tag and publishes ready-to-use binaries for all target platforms.

## Goals / Non-Goals

**Goals:**
- Trigger a release build on push of a `v*.*.*` tag
- Cross-platform builds: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64, freebsd/amd64, freebsd/arm64
- Publish a GitHub Release with attached binaries and auto-generated release notes from git log
- Embed the version string into the binary via `-ldflags`

**Non-Goals:**
- Publishing to package registries (brew, apt, scoop, etc.)
- Docker images
- Binary signing / notarization
- Automatic tag creation (tags are pushed manually)

## Decisions

### Native `go build` with platform matrix vs GoReleaser

**Decision**: native `go build` + GitHub Actions matrix + `gh release create`

GoReleaser is a powerful tool with templating, signing, brew-tap support, etc., but it requires an additional config file (`.goreleaser.yaml`) and an external dependency. The project is small and requirements are straightforward — a GOOS/GOARCH matrix fully covers the need without extra tooling. GoReleaser can be adopted later if the release process grows more complex.


### Gating releases on lint and tests

**Decision**: the release workflow does not depend on the CI workflow file. Since it triggers on a tag (not a branch), a direct `workflow_run` dependency is possible but adds latency and complexity. Instead, lint and test steps are run inside the release workflow as gates before the build matrix starts.

**Alternative considered**: `workflow_run` trigger — rejected because it introduces coupling between two separate trigger events and makes the release pipeline harder to reason about.

### Version embedding

Version is embedded and debug info is stripped via `-ldflags "-s -w -X main.version=${{ github.ref_name }}"`. `-s` removes the symbol table, `-w` removes DWARF debug info — together they reduce binary size significantly with no runtime impact.

### Artifact naming

Format: `claude-openai-proxy-<os>-<arch>` (with `.exe` suffix for Windows). Raw binaries are uploaded directly to the GitHub Release without any archiving.

## Risks / Trade-offs

- **Duplicated lint/test config** in release workflow → minor duplication; can be extracted into a reusable workflow later if drift becomes a problem
- **Matrix build time** increases total workflow duration → acceptable, Go cross-compilation is fast (~1–2 min per platform)
- **Tag pushed before CI passes on the branch** → mitigation: document in release process that tags should only be pushed after CI is green on master

## Migration Plan

1. Add `.github/workflows/release.yml`
2. Validate on a test tag (`v0.0.1-test`)
3. Delete test tag after validation
4. Document the release process in README (out of scope for this change)