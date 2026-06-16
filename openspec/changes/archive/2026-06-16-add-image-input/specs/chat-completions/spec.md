## MODIFIED Requirements

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

## ADDED Requirements

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
