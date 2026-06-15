## Why

Today the only way to install `claude-openai-proxy` is to download a raw per-platform binary from the GitHub release page and place it on `PATH` manually. There is no first-class installation path for macOS/Linux users. A Homebrew tap gives them a one-line install (`brew install`) plus automatic upgrades, which is the expected distribution channel for a CLI service of this kind.

## What Changes

- Introduce a dedicated Homebrew **tap** GitHub repository (`homebrew-tap`, separate from this code repo) that hosts the generated **formula** (`Formula/claude-openai-proxy.rb`).
- Users install via `brew tap kpod13/tap && brew install claude-openai-proxy` and upgrade via `brew upgrade`.
- Extend the existing `release.yml` workflow so it packages cross-platform `.tar.gz`/`.zip` archives, generates `checksums.txt`, creates the GitHub Release, and regenerates + pushes the formula to the tap repo on every `v*.*.*` tag (no GoReleaser — we keep the current `go build` matrix).
- Add a formula template + a small `scripts/update-formula.sh` (no application code changes) and the CI secret/token wiring needed to push to the tap repo.
- Document the install flow in the README.
- The release artifact format changes from raw binaries to versioned `.tar.gz`/`.zip` archives + checksums (needed so the formula can pin a `sha256`).

This change is **configuration and packaging only**; it does not modify the Go source of the proxy.

## Capabilities

### New Capabilities
- `homebrew-distribution`: Installation and upgrade of the proxy through a Homebrew tap — tap repository layout, formula contents/requirements, the user-facing `brew tap`/`brew install`/`brew upgrade` flow, and a formula `test` block.

### Modified Capabilities
- `release`: The release pipeline switches from manually building raw per-platform binaries to GoReleaser-produced archives + checksums, and additionally publishes the Homebrew formula to the tap repository as part of the tagged release.

## Impact

- **New repo**: `kpod13/homebrew-tap` (tap), containing `Formula/claude-openai-proxy.rb` (auto-generated/updated by GoReleaser).
- **This repo**: new `scripts/update-formula.sh` + formula template; extended `release` job in `.github/workflows/release.yml`; README install section.
- **CI secrets**: a token (PAT or fine-grained) with write access to the tap repo, exposed to the release workflow for the formula push.
- **Artifacts**: release assets change shape (archives + `checksums.txt` instead of bare binaries) — anyone scripting downloads of the old raw-binary asset names is affected.
- **Dependencies**: no new release-toolchain dependency (reuses the existing `go build` matrix); end users gain a Homebrew dependency (optional, install path only).
