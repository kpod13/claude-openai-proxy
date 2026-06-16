## Purpose

Defines the OpenAI-compatible `POST /v1/chat/completions` endpoint: how requests (text and multimodal) are translated into `claude` CLI invocations and how streaming and non-streaming responses are shaped.
## Requirements
### Requirement: Chat completions endpoint
The server SHALL expose `POST /v1/chat/completions` accepting an OpenAI-compatible request body and returning an OpenAI-compatible response.

#### Scenario: Non-streaming completion
- **WHEN** a POST request is sent to `/v1/chat/completions` with `"stream": false` (or field absent)
- **THEN** the server invokes `claude --print --output-format json` and returns a JSON response matching the OpenAI `ChatCompletion` object schema with HTTP 200

#### Scenario: Streaming completion
- **WHEN** a POST request is sent to `/v1/chat/completions` with `"stream": true`
- **THEN** the server invokes `claude --print --output-format stream-json --verbose`, streams SSE chunks matching the OpenAI `ChatCompletionChunk` schema, and terminates with `data: [DONE]`

### Requirement: Message serialization
The server SHALL convert the OpenAI `messages` array into input for the `claude` CLI passed via stdin. A message's `content` MAY be a plain string or an array of content parts (`text` and `image_url`). Text-only conversations are serialized into a single prompt string (text input); conversations containing image parts are serialized into stream-json input carrying content blocks.

#### Scenario: System message present
- **WHEN** the messages array includes a `system` role message
- **THEN** its content is prepended as `[System]: <content>` in the assembled prompt

#### Scenario: Multi-turn conversation
- **WHEN** the messages array contains alternating `user` and `assistant` messages
- **THEN** each message is included in order as `[User]: <content>` or `[Assistant]: <content>`

#### Scenario: String content still accepted
- **WHEN** a message's `content` is a plain JSON string
- **THEN** it is treated as a single text part, exactly as before this change

#### Scenario: Unsupported message content type
- **WHEN** a message contains a content part that is neither text nor `image_url` (e.g. `input_audio`)
- **THEN** the server returns HTTP 400 with a descriptive error

### Requirement: Model selection
The server SHALL pass the resolved model ID to `claude --model <id>` for each request.

#### Scenario: Valid model in request
- **WHEN** the request body contains a recognized `model` value
- **THEN** the CLI is invoked with `--model <resolved-full-id>`

#### Scenario: Invalid model in request
- **WHEN** the request body contains an unrecognized `model` value
- **THEN** the server returns HTTP 400 before invoking the CLI

### Requirement: Usage reporting
The server SHALL populate the `usage` field in non-streaming responses using token counts from the `claude` CLI JSON output.

#### Scenario: Token counts available
- **WHEN** the CLI returns a `usage` block with `input_tokens` and `output_tokens`
- **THEN** these are mapped to `prompt_tokens` and `completion_tokens` in the OpenAI response

### Requirement: Image input in chat completions
The server SHALL accept OpenAI `image_url` content parts and forward them to the `claude` CLI using `--input-format stream-json`, so requests that include images return an OpenAI-compatible completion (streaming and non-streaming) rather than an error.

#### Scenario: Base64 data URI image
- **WHEN** a user message contains an `image_url` part whose URL is a `data:<mime>;base64,<...>` URI
- **THEN** the server decodes it into a base64 image content block and includes it in the stream-json message sent to the CLI

#### Scenario: Remote URL image
- **WHEN** a user message contains an `image_url` part whose URL is an `http(s)` URL
- **THEN** the server includes it as a URL image content block (the model backend fetches it) without the proxy downloading the bytes

#### Scenario: Mixed text and multiple images
- **WHEN** a user message contains a `text` part and two `image_url` parts
- **THEN** all parts are forwarded in order within a single message's content blocks

#### Scenario: Malformed data URI
- **WHEN** an `image_url` part contains a `data:` URI that cannot be parsed or base64-decoded
- **THEN** the server returns HTTP 400 with a descriptive error

### Requirement: Permission flags on claude invocation
The server SHALL include the configured permission policy flags (`--permission-mode`, `--allowedTools`, `--disallowedTools`, `--add-dir`) when invoking `claude` for `/v1/chat/completions`, for both streaming and non-streaming requests and for both text-only and image-bearing requests. With the safe default policy, no permission flags beyond the implicit `default` mode are added, leaving the invocation unchanged from prior behavior.

#### Scenario: Allowlisted tool call succeeds headlessly
- **WHEN** the operator has configured `allowed_tools` covering the tool a request triggers
- **THEN** the `claude` invocation carries the allowlist and the tool runs without blocking on an interactive prompt

#### Scenario: Default policy leaves invocation unchanged
- **WHEN** no permission policy is configured
- **THEN** the `claude` invocation contains no allowlist and no permission-bypass flags

