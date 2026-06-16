## Why

OpenAI-compatible clients send images through multimodal `content` arrays (`{"type": "image_url", ...}` parts). The proxy currently accepts only a string `content` and explicitly returns HTTP 400 for any non-text part (see the `chat-completions` spec, "Unsupported message content type"). That blocks every vision use case, even though both the OpenAI API and the underlying Claude models support image input.

## What Changes

- Accept OpenAI multimodal `content` on `POST /v1/chat/completions`: a `content` value may be a plain string (as today) **or** an array of parts (`text` and `image_url`).
- Forward images to the `claude` CLI using `--input-format stream-json`: assemble a user message whose content includes Anthropic image blocks, written to stdin (instead of the current plain-text prompt argument). The existing text-only path is preserved when no image parts are present.
- Decode `image_url` from both base64 `data:` URIs and `http(s)` URLs; support multiple images per message.
- Return HTTP 400 only for genuinely unsupported part types (e.g. `input_audio`), not for images.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `chat-completions`: message content may now be multimodal (text + images); image parts are forwarded to the CLI via stream-json instead of being rejected. The "unsupported content" rule narrows to non-text/non-image parts only.

## Impact

- **Code**: `internal/proxy/types.go` (`Message.Content` becomes polymorphic — string or `[]ContentPart`, via custom `UnmarshalJSON`); `internal/proxy/claude.go` (stream-json invocation path carrying image blocks); `internal/proxy/handler.go` (content parsing, validation, error mapping). Tests in the corresponding `_test.go` files.
- **Behavior (was BREAKING-ish)**: requests with `image_url` previously got 400; they will now succeed. Plain-string requests are unaffected.
- **Dependency**: relies on `claude --print --input-format stream-json` accepting Anthropic image content blocks — to be confirmed during implementation.
- **Issue**: [#18](https://github.com/kpod13/claude-openai-proxy/issues/18).
