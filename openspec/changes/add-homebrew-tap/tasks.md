## 1. Create the tap repository

- [x] 1.1 Create a new GitHub repository `kpod13/homebrew-tap` (public)
- [x] 1.2 Initialize it with a README describing the tap and a `Formula/` directory placeholder
- [x] 1.3 Verify `brew tap kpod13/tap` resolves to the repo (short-form prefix works)

## 2. Provision the cross-repo token

- [x] 2.1 Create a fine-grained PAT scoped to `kpod13/homebrew-tap` with contents:write
- [x] 2.2 Add it as a secret (e.g. `HOMEBREW_TAP_TOKEN`) in this repo's Actions settings
- [x] 2.3 Document token purpose and rotation steps in the design/README

## 3. Add the formula template and render script

- [x] 3.1 Add a Homebrew formula template (`scripts/claude-openai-proxy.rb.tmpl`) covering macOS arm/intel and Linux arm/intel `url` + `sha256`, `version`, `bin.install`, and a `test` block calling `--version`
- [x] 3.2 Add `scripts/update-formula.sh` that takes the version + artifacts dir, reads each archive's sha256 from `checksums.txt`, and renders `Formula/claude-openai-proxy.rb`
- [x] 3.3 Validate the script locally against a snapshot build (render formula, no publish)

## 4. Extend the release workflow

- [x] 4.1 Keep the lint → test (≥25% coverage) gate ahead of build
- [x] 4.2 Update the build matrix to package each binary into a `.tar.gz` (Unix) / `.zip` (Windows) archive named `claude-openai-proxy_<version>_<os>_<arch>`
- [x] 4.3 In the release job, generate `checksums.txt` over all archives and create the GitHub Release with archives + checksums via `gh release create`
- [x] 4.4 Add a step that runs `scripts/update-formula.sh`, then checks out `kpod13/homebrew-tap` (using `HOMEBREW_TAP_TOKEN`) and commits/pushes the regenerated formula
- [x] 4.5 Confirm the workflow still triggers only on `v*.*.*` tags and not on branch pushes

## 5. Verify end to end

- [ ] 5.1 Push a test/pre-release tag and confirm the GitHub Release has archives + `checksums.txt`
- [ ] 5.2 Confirm `Formula/claude-openai-proxy.rb` is committed/updated in the tap repo with correct url, sha256, version
- [ ] 5.3 Run `brew install kpod13/tap/claude-openai-proxy` and confirm the binary is on `PATH`
- [ ] 5.4 Run `brew test claude-openai-proxy` and confirm it passes
- [ ] 5.5 Confirm `brew upgrade` picks up a subsequent release

## 6. Documentation

- [x] 6.1 Add a Homebrew install section to the README (`brew tap` + `brew install`)
- [x] 6.2 Note the new release asset shape (archives + checksums) for users who scripted raw-binary downloads
