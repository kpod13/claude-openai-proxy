## ADDED Requirements

### Requirement: List models endpoint
The server SHALL expose `GET /v1/models` returning all discovered models in OpenAI-compatible format.

#### Scenario: Models available
- **WHEN** a GET request is sent to `/v1/models`
- **THEN** the server returns HTTP 200 with a JSON body matching the OpenAI `ModelList` schema (`object: "list"`, `data: [...]`)

#### Scenario: Each model object structure
- **WHEN** the models list is returned
- **THEN** each entry contains `id` (full Claude model ID), `object: "model"`, `created` (Unix timestamp), and `owned_by: "anthropic"`
