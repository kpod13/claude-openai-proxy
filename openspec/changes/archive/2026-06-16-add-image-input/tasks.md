## 1. Confirm the CLI image interface (spike)

- [x] 1.1 Determine the exact `claude --print --input-format stream-json` stdin envelope (message wrapper, single vs multiple messages)
- [x] 1.2 Verify a base64 image content block is accepted and produces a normal completion
- [x] 1.3 Verify whether a `url` image source works or images must be fetched and inlined; record the decision
- [x] 1.4 Confirm both `--output-format json` and `stream-json` work alongside stream-json input

## 2. Parse multimodal content

- [x] 2.1 Introduce a `ContentPart` type and make `Message.Content` polymorphic via custom `UnmarshalJSON` (accept string or array)
- [x] 2.2 Normalize a plain string into a single text part
- [x] 2.3 Map `image_url` parts into an internal image representation (data URI vs http(s) URL)
- [x] 2.4 Return 400 for unsupported part types and malformed `data:` URIs

## 3. Forward images to the CLI

- [x] 3.1 Build the stream-json stdin payload with text + image content blocks (per the spike)
- [x] 3.2 Invoke `claude --print --input-format stream-json` when any message has image parts; keep the text path otherwise
- [x] 3.3 Ensure both non-streaming and streaming responses work with image input

## 4. Tests

- [x] 4.1 Unit tests for `Content` unmarshaling (string, text array, text+image, unsupported part)
- [x] 4.2 Handler tests: data URI image, remote URL image, multiple images, malformed data URI → 400
- [x] 4.3 `go test ./...` and `go vet ./...`

## 5. Docs

- [x] 5.1 Note image-input support (multimodal `content`) in the README
