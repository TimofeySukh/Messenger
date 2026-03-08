# Client File Map

## `cmd/client/main.go`

- Loads config from `~/.messenger/config.toml`.
- Resolves flags and interactive prompts.
- Starts session runner with cancelable context.
- Persists server/username/alias updates.

## `internal/protocol/protocol.go`

- Shared protocol contract on the client side.
- JSONL message codec and validation.

## `internal/protocol/protocol_test.go`

- Validates client encode/decode compatibility.

## `internal/crypto/crypto.go`

- Generates AES-256 keys.
- Encrypt/decrypt helpers with AES-GCM.
- Base64 key validation.

## `internal/config/config.go`

- Loads and saves `config.toml`.
- Supports `[aliases]` for room shortcuts.

## `internal/history/history.go`

- Stores local encrypted history per room.
- Reads decrypted history entries.
- Exports plaintext transcript on demand.

## `internal/ui/ui.go`

- Contains terminal rendering primitives for session startup output.

## `internal/session/session.go`

- Connects and performs protocol handshake.
- Handles reconnect attempts after read/write failures.
- Tracks pending messages and retries until ack.
- Processes slash commands and incoming server events.
- Emits mention bell for `@username` messages.

## `internal/session/session_test.go`

- Validates `/leave` command path.
- Validates reconnect path with a mock protocol server.
