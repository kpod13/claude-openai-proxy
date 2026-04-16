## 1. Release Workflow

- [ ] 1.1 Create `.github/workflows/release.yml` with trigger on `v*.*.*` tag push
- [ ] 1.2 Add `lint` job to release workflow (reuse `.golangci.yml` config)
- [ ] 1.3 Add `test` job to release workflow with `needs: lint` and coverage check
- [ ] 1.4 Add `build` job with platform matrix (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64, freebsd/amd64, freebsd/arm64) gated on `needs: test`
- [ ] 1.5 Configure `go build` in matrix with `-ldflags "-s -w -X main.version=${{ github.ref_name }}"` to strip debug info and embed version
- [ ] 1.6 Name binaries using the pattern `claude-openai-proxy-<os>-<arch>` (`.exe` suffix for Windows)
- [ ] 1.7 Upload raw binaries as workflow artifacts in the build job

## 2. GitHub Release

- [ ] 2.1 Add `release` job with `needs: build` and `permissions: contents: write`
- [ ] 2.2 Download all platform archives from workflow artifacts
- [ ] 2.3 Create GitHub Release using `gh release create ${{ github.ref_name }}` with `--generate-notes` and all archives attached

## 3. Validation

- [ ] 3.1 Push a test tag (`v0.0.1-test`) and verify the workflow runs end-to-end
- [ ] 3.2 Confirm all eight platform binaries appear on the GitHub Release page
- [ ] 3.3 Confirm release notes are auto-generated
- [ ] 3.4 Delete the test tag and release after validation
