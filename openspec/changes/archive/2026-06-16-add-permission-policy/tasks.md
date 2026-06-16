## 1. Config

- [x] 1.1 Add a `Permission` struct (`Mode`, `AllowedTools`, `DisallowedTools`, `AddDirs`) and a `Permission` field to `Config` in `internal/config/config.go` with YAML tags
- [x] 1.2 Set the safe default in defaults: `Mode: "default"`, all lists empty (no allowlist, no bypass)
- [x] 1.3 Validate the whole permission block at startup (before binding the listener): `Mode` must be one of the `claude` `--permission-mode` choices (`acceptEdits`, `auto`, `bypassPermissions`, `default`, `dontAsk`, `plan`); each `AllowedTools`/`DisallowedTools` entry must match the tool-spec format `^[A-Za-z][A-Za-z0-9_]*(\([^)]+\))?$` after trimming; each `AddDirs` entry must be a non-blank path; and no entry in any list may begin with `-`. Return an informative startup error (naming the offending value) otherwise
- [x] 1.4 Add config tests: permission block parsed, block absent uses safe default, each supported mode accepted, invalid mode rejected, well-formed tool specs accepted (`Write`, `Bash(git *)`, `mcp__server__tool`), malformed tool spec rejected (unclosed rule, leading digit), blank entry rejected, leading-`-` entry rejected

## 2. CLI flag assembly

- [x] 2.1 Add a `PermissionArgs(p config.Permission) []string` helper in `internal/proxy/claude.go` that emits `--permission-mode` (only when non-default), `--allowedTools`, `--disallowedTools`, and `--add-dir` flags; empty lists/default mode emit nothing
- [x] 2.2 Thread the permission policy into the claude invocations via `BlockingRunner`/`BlockingImagesRunner`/`StreamingRunner`/`StreamingImagesRunner` constructors that append the flags to each arg list (default `Run*` funcs kept as no-perm fallbacks for the existing 3-arg runner seam)
- [x] 2.3 Wire the policy into the handler from the loaded `*config.Config` in `cmd/claude-openai-proxy/main.go` (the handler delegates to injected runners, so the perm-bound runners are set there and then debug-wrapped when `--verbose`)
- [x] 2.4 Add tests asserting the emitted args for: default policy (no permission flags), allowlist, mode selection, add_dirs, and consistency across all four invocation paths

## 3. Docs

- [x] 3.1 Add a Permissions section to `README.md` describing the `permission` config block, the safe default, and the `0.0.0.0` + `bypassPermissions` security warning
- [x] 3.2 Rewrite `CLAUDE.md` to document the actual Go proxy (architecture, build/test/lint via Makefile, package layout, how `claude` is invoked) instead of the outdated OpenSpec-template content

## 4. Verify

- [x] 4.1 Run `make lint` and `make test`; fix any findings (lint: 0 issues; all packages pass)
- [x] 4.2 Verified startup behavior end-to-end: invalid mode and flag-like entry fail fast (exit 1, informative error, before binding); a valid `acceptEdits` + allowlist config passes validation and proceeds to model discovery. Flag emission across all four runner paths is covered by `TestRunnersEmitPermissionFlags` / `TestDefaultRunnersEmitNoPermissionFlags` (a full live tool-triggering request needs claude auth and is out of scope for this check)
