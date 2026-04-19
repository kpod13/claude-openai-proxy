## Context

No README exists. The project is a Go HTTP proxy that translates OpenAI-compatible API calls to Claude CLI commands. It has installation, config, and rate limiting features that need to be documented.

## Goals / Non-Goals

**Goals:**
- Single `README.md` at repo root covering all user-facing features.

**Non-Goals:**
- Separate docs site or multiple markdown files.
- API reference (covered by OpenAI docs).

## Decisions

**Use plain GitHub-flavoured Markdown** — no doc generator needed for a single file.

**Include a quick-start code block** — users should be able to copy-paste their way to a running server in under a minute.

**Document rate limiting config** — it's opt-in and invisible without documentation.
