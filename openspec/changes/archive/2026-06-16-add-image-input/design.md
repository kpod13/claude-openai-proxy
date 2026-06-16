## Context

`POST /v1/chat/completions` deserializes each message into `Message{ Role, Content string }` and serializes the conversation into a single text prompt passed to `claude --print` (text input). Image input is explicitly rejected with HTTP 400.

The `claude` CLI supports `--input-format stream-json` (with `--print`), which reads structured message(s) from stdin. This is the seam for delivering Anthropic-style content blocks — including image blocks — to the model without an interactive session. OpenAI's wire format for images is the `image_url` content part (base64 `data:` URI or `http(s)` URL).

## Goals / Non-Goals

**Goals:**
- Accept OpenAI multimodal `content` (string or array of `text`/`image_url` parts).
- Forward images to Claude and get a normal OpenAI-shaped completion back (streaming and non-streaming).
- Preserve the existing text-only path unchanged.

**Non-Goals:**
- Image *output* / generation.
- Other part types (`input_audio`, files) — still 400.
- Re-encoding, resizing, or validating image bytes beyond what's needed to build the block.

## Decisions

### D1: Polymorphic `Content` via custom `UnmarshalJSON`
Change `Message.Content` from `string` to a small type that unmarshals either a JSON string or an array of parts into a normalized `[]ContentPart{ Type, Text, ImageURL }`. A plain string becomes a single text part. This keeps the public JSON contract OpenAI-compatible and isolates the polymorphism.
- *Alternative*: `json.RawMessage` + ad-hoc branching in the handler. Rejected — leaks parsing into the handler and is harder to test.

### D2: Use `--input-format stream-json` only when images are present
When every message is text-only, keep the current text-prompt invocation (well-tested, lowest risk). When any message carries an image part, switch to building a stream-json stdin payload with content blocks.
- *Alternative*: always use stream-json. Rejected for now — larger blast radius on the common text path; revisit if maintaining two paths proves annoying.

### D3: Map `image_url` to the right Anthropic block by source
- `data:<mime>;base64,<...>` → base64 image block (`source: {type: "base64", media_type, data}`).
- `http(s)://...` → URL image block (`source: {type: "url", url}`), letting Claude's backend fetch it.
- *Rationale*: forwarding the URL avoids the proxy fetching arbitrary URLs (SSRF surface) and avoids buffering remote bytes. The proxy only base64-decodes inline data URIs it was already given.

### D4: Validation and errors
Unknown/invalid `data:` URIs → 400. Unsupported part types (`input_audio`, etc.) → 400 with a descriptive message. An empty/whitespace `text` alongside images is allowed.

## Risks / Trade-offs

- **stream-json input schema uncertainty** → The exact envelope `claude --print --input-format stream-json` expects (message wrapper, whether tools must be disabled, multi-message vs single) is not yet pinned. Mitigation: a spike task confirms the format against the installed CLI before wiring the handler; the design isolates this in `claude.go`.
- **Two invocation paths (text vs stream-json)** → some duplication / divergence risk. Mitigation: share serialization helpers; cover both paths in tests.
- **Large base64 over stdin** → big images inflate the payload. Mitigation: stream to the CLI's stdin (already how prompts are passed); no extra buffering beyond the request body.
- **URL images depend on backend fetchability** → a URL Claude can't reach fails the turn. Acceptable; mirrors OpenAI behavior for unreachable URLs.

## Open Questions

- Exact `--input-format stream-json` message envelope and whether `--output-format json`/`stream-json` both work with it (streaming responses).
- Does the CLI accept the `url` image source, or must URLs be fetched and inlined by the proxy? (Falls back to fetch-and-inline if not.)
- Image size/count limits worth enforcing proxy-side.
