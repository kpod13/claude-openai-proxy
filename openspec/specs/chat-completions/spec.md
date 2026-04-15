### Requirement: Chat completions endpoint
The server SHALL expose `POST /v1/chat/completions` accepting an OpenAI-compatible request body and returning an OpenAI-compatible response.

#### Scenario: Non-streaming completion
- **WHEN** a POST request is sent to `/v1/chat/completions` with `"stream": false` (or field absent)
- **THEN** the server invokes `claude --print --output-format json` and returns a JSON response matching the OpenAI `ChatCompletion` object schema with HTTP 200

#### Scenario: Streaming completion
- **WHEN** a POST request is sent to `/v1/chat/completions` with `"stream": true`
- **THEN** the server invokes `claude --print --output-format stream-json --verbose`, streams SSE chunks matching the OpenAI `ChatCompletionChunk` schema, and terminates with `data: [DONE]`

### Requirement: Message serialization
The server SHALL convert the OpenAI `messages` array into a single prompt string passed to the `claude` CLI via stdin.

#### Scenario: System message present
- **WHEN** the messages array includes a `system` role message
- **THEN** its content is prepended as `[System]: <content>` in the assembled prompt

#### Scenario: Multi-turn conversation
- **WHEN** the messages array contains alternating `user` and `assistant` messages
- **THEN** each message is included in order as `[User]: <content>` or `[Assistant]: <content>`

#### Scenario: Unsupported message content type
- **WHEN** a message contains non-text content (e.g., image_url)
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
