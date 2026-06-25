## 1. Default tool lists in config

- [x] 1.1 Add package-level `defaultAllowedTools` (`WebSearch`, `WebFetch`) and `defaultDisallowedTools` (`Artifact`, `Bash`, `Edit`, `ExitPlanMode`, `Monitor`, `NotebookEdit`, `PowerShell`, `ShareOnboardingGuide`, `Skill`, `Workflow`, `Write`) constants/vars in `internal/config/config.go`.
- [x] 1.2 In `(*Config).validate()`, before validating the tool specs, substitute the built-in default into each empty list independently (use copies of the shared slices so in-place trimming never mutates them). Also run the no-config-file default through `validate()` in `Load()` so the "policy absent" path gets the defaults.
- [x] 1.3 Update the `Permission` / safe-default doc comments in `config.go` to describe the new per-section defaults.

## 2. Tests

- [x] 2.1 Add config tests covering: both sections empty → both defaults applied; only `allowed_tools` set → custom allowed + default disallowed; only `disallowed_tools` set → default allowed + custom disallowed; both set → no defaults; defaults pass validation.
- [x] 2.2 Verify shared default slices are not mutated across loads (no leaking trimmed/aliased state between configs).

## 3. Documentation

- [x] 3.1 Update `README.md` to document the new default `allowed_tools` / `disallowed_tools` and the per-section override behavior.

## 4. Verification

- [x] 4.1 Run `make test` and `make lint`; ensure both pass.
