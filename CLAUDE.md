# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Repo Is

A **configuration-only** Claude Code extension that integrates the [OpenSpec](https://openspec.dev) specification-driven development workflow. It contains no application code — only skill and command definitions.

**Dependency**: The `openspec` CLI must be installed and available on `PATH`.

## No Build System

There are no build scripts, package.json, Makefile, or test framework. All files are markdown definitions interpreted by Claude Code at runtime.

## Slash Commands and Skills

The four workflow commands are defined in `.claude/commands/opsx/` and backed by skills in `.claude/skills/`:

| Command | Skill | Purpose |
|---|---|---|
| `/opsx:propose` | `openspec-propose` | Create a change directory + all artifacts |
| `/opsx:explore` | `openspec-explore` | Think-partner mode, no code changes |
| `/opsx:apply` | `openspec-apply-change` | Implement tasks from a change |
| `/opsx:archive` | `openspec-archive-change` | Validate and move change to archive |

The command files (`.claude/commands/`) are near-identical copies of the skill files (`.claude/skills/`). The canonical source of truth for skill logic is in the `SKILL.md` files.

## OpenSpec Workflow

Changes live in `openspec/changes/<name>/` and follow this lifecycle:

```
propose → [explore] → apply → archive
```

Each change directory contains artifacts created in dependency order:
- `proposal.md` — what & why
- `design.md` — how (depends on proposal)
- `tasks.md` — implementation checklist (depends on design)
- `specs/<capability>/spec.md` — detailed requirements (optional)

Archived changes are moved to `openspec/changes/archive/YYYY-MM-DD-<name>/`.

Main capability specs live in `openspec/specs/`.

## Key CLI Commands Used by Skills

```bash
openspec new change "<name>"
openspec status --change "<name>" --json
openspec instructions <artifact-id> --change "<name>" --json
```

## Skill Authoring Conventions

When editing skill or command files:

- `context` and `rules` fields from `openspec instructions` output are constraints for the AI, **never** content to include in artifact files
- Tasks use `- [ ]` / `- [x]` checkboxes; mark complete immediately, not in batch
- Skills pause and use `AskUserQuestion` on ambiguity rather than guessing
- `applyRequires` in the schema defines the minimum artifact set needed before implementation starts
- Explore skill is read-only — it investigates code but never writes application code
