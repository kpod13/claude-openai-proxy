## 1. Preparation

- [x] 1.1 Run `go test -coverprofile=coverage.out ./...` locally and check total coverage to set an appropriate minimum threshold

## 2. Workflow file

- [x] 2.1 Create `.github/workflows/ci.yml` with triggers: `push` to `master` and `pull_request` targeting `master`
- [x] 2.2 Add `lint` job using `golangci/golangci-lint-action` pinned to a major version tag
- [x] 2.3 Add `test` job with `needs: lint`, running on `ubuntu-latest` with the pinned Go version
- [x] 2.4 In the `test` job: run `go test -coverprofile=coverage.out ./...`
- [x] 2.5 In the `test` job: add a shell step that extracts total coverage and fails if below the minimum threshold

## 3. Makefile

- [x] 3.1 Add `ci` to the `.PHONY` list in `Makefile`
- [x] 3.2 Add `ci` target: `act push -P ubuntu-latest=catthehacker/ubuntu:act-latest`

## 4. Verification

- [x] 4.1 Validate workflow YAML syntax (`yamllint` or `actionlint` if available, otherwise visual inspection)
- [x] 4.2 Confirm `go test ./...` passes locally
- [x] 4.3 Confirm `golangci-lint run ./...` passes locally
- [x] 4.4 Run `make ci` and confirm it exits 0
