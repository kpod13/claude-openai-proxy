## Context

`claude-openai-proxy` is a single-binary Go service. The current release pipeline (`.github/workflows/release.yml`) lints, tests, then runs a hand-written matrix of `go build` jobs that upload **raw, unarchived binaries** named `claude-openai-proxy-<os>-<arch>` to a GitHub Release created with `gh release create`. There are no archives and no checksum file.

Homebrew is the conventional install channel for a CLI/service on macOS (and works on Linux via Linuxbrew). For a non-GUI, open-source binary the correct Homebrew artifact is a **formula** (Ruby file under `Formula/`), not a cask. A formula must pin each downloadable archive to a `sha256`, which means the release must publish stable archive URLs and a checksums file.

A Homebrew tap is just a GitHub repo named `homebrew-<suffix>`; the `homebrew-` prefix lets users type the short form `brew tap kpod13/<suffix>`. Owner is `kpod13` (matches the Go module path `github.com/kpod13/claude-openai-proxy`).

## Goals / Non-Goals

**Goals:**
- A one-line install + automatic upgrades on macOS/Linux via Homebrew.
- The formula is regenerated and pushed to the tap automatically on every `v*.*.*` tag — no manual `sha256` editing.
- Release artifacts carry checksums so installs are verifiable.
- No changes to the proxy's Go source.

**Non-Goals:**
- Submitting to `homebrew/core` (a personal tap is sufficient and avoids core's acceptance bar).
- Building a cask or any GUI packaging.
- Windows packaging (Scoop/winget/MSI) — out of scope; archives still build for Windows but distribution stays as release assets.
- Code signing / notarization of the macOS binary (tracked as an open question; not required for a tap formula).

## Decisions

### D1: Tap is a separate `kpod13/homebrew-tap` repository
A tap MUST live in its own repo named `homebrew-*`. We use `homebrew-tap` so the user command is `brew tap kpod13/tap`. The formula lives at `Formula/claude-openai-proxy.rb`. The formula name (`claude-openai-proxy`) becomes the `brew install` argument.
- *Alternative considered*: `homebrew-claude-openai-proxy` (one tap per tool). Rejected — a single `homebrew-tap` can host future tools and is the common pattern.

### D2: Formula, not cask
The artifact is an open-source CLI binary → formula. Casks are intended for GUI/closed-source apps and are macOS-only; a formula additionally works on Linuxbrew and supports a `test do` block and `livecheck`.
- *Alternative considered*: cask wrapping the binary (GoReleaser supports `homebrew_casks`). Rejected per D2 rationale; formula is the idiomatic choice for a CLI.

### D3: Extend the existing GitHub Actions workflow (no GoReleaser)
Keep the hand-rolled `.github/workflows/release.yml` and its `go build` matrix; add the packaging and formula-publishing steps ourselves. Concretely:
- the build matrix packages each binary into a `.tar.gz` (Unix) / `.zip` (Windows) archive,
- the release job collects all archives, generates `checksums.txt`, and creates the GitHub Release with `gh release create` and generated notes,
- a `scripts/update-formula.sh` renders `Formula/claude-openai-proxy.rb` from a template — filling `version` and the per-platform `url` + `sha256` (macOS arm/intel, Linux arm/intel) read from the built archives/checksums — and the workflow commits/pushes it to `kpod13/homebrew-tap`.
- *Alternative considered*: GoReleaser with a `brews:` block. Rejected — it is a heavy external dependency that mostly duplicates the existing build matrix, and GoReleaser has **deprecated** its formula (`brews`) path in favor of macOS-only `homebrew_casks`. Writing the formula ourselves keeps a true cross-platform (macOS + Linuxbrew) **formula** with full control and no deprecation risk.
- *Alternative considered*: a prebuilt action such as `dawidd6/action-homebrew-bump-formula`. Rejected — geared toward single-source-URL formulae; awkward for a multi-platform binary formula.

### D4: Tap push uses a dedicated cross-repo token
The default `GITHUB_TOKEN` cannot push to a *different* repo. The release workflow exposes a secret (PAT or fine-grained token scoped to `kpod13/homebrew-tap` contents:write) used only by the step that checks out the tap repo and pushes the regenerated formula. The GitHub Release itself still uses the built-in `GITHUB_TOKEN`.

### D5: Lint/test gate is preserved
The existing lint → test → (≥25% coverage) gate stays in front of the GoReleaser job; release only runs if both pass. This keeps the `release` spec's gating requirement intact.

## Risks / Trade-offs

- **Artifact shape change (BREAKING for scripted downloads)** → Release assets become archives + `checksums.txt` instead of raw `claude-openai-proxy-<os>-<arch>` files. Mitigation: document the new asset names in the README; the binary inside each archive keeps the canonical name.
- **Cross-repo token leak / expiry** → A PAT with write access to the tap is sensitive and can expire, silently breaking formula publishing. Mitigation: use a fine-grained token limited to the tap repo's contents; document rotation; failures surface as a failed release job.
- **Tap repo must exist before first tagged release** → GoReleaser push fails if `kpod13/homebrew-tap` is absent. Mitigation: create and initialize the tap repo (with a README) as an explicit task before the first release.
- **macOS Gatekeeper on unsigned binary** → Users may hit a quarantine warning. Mitigation: formula installs to a Homebrew prefix where quarantine is typically not applied to formula binaries; revisit signing if reports arise (open question).
- **Version flag contract** → `test do` asserts `--version` prints the tag; if the proxy's flag differs, the formula test fails. Mitigation: confirm the actual `--version`/`version` invocation when authoring the formula test.

## Migration Plan

1. Create `kpod13/homebrew-tap` (empty repo + README).
2. Add `scripts/update-formula.sh` + a formula template; extend `release.yml` to package archives, generate checksums, and push the formula; add the tap token secret.
3. Validate the packaging/formula-render steps with a dry run (e.g. `act` or a throwaway pre-release tag) — no publish to the real tap.
4. Cut a test tag (e.g. pre-release) and confirm the formula lands in the tap and `brew install kpod13/tap/claude-openai-proxy` works.
5. Update README with install instructions.
- *Rollback*: the old `release.yml` is recoverable from git history; reverting the workflow restores raw-binary releases. The tap repo can be left in place (harmless) or deleted.

## Open Questions

- Exact `--version` invocation/output to assert in the formula `test` block. (cobra prints `claude-openai-proxy version <v>`.)
- Whether to add a `livecheck` block / macOS code signing now or defer.
- Tap owner namespace confirmation (`kpod13`) and whether other future tools should share this tap.
