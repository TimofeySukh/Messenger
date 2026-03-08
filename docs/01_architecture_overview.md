# Architecture Overview

## Runtime Topology

- The server accepts TCP connections and coordinates room membership.
- The client encrypts/decrypts chat payloads locally.
- The server sees encrypted payloads, metadata, and routing fields only.

## Layering

### Server

- `cmd/server` - process bootstrap and signal handling.
- `internal/protocol` - JSONL typed protocol encoding/decoding.
- `internal/room` - room manager, join limiter, token generation, fanout.
- `internal/transport` - TCP lifecycle, handshake, ping/pong, command routing.

### Client

- `cmd/client` - CLI entrypoint, flags, config loading, prompts.
- `internal/config` - `~/.messenger/config.toml` read/write and aliases.
- `internal/protocol` - same typed protocol model and codec.
- `internal/crypto` - AES-256-GCM and key validation.
- `internal/session` - reconnect loop, ack/retry, slash command dispatch.
- `internal/history` - encrypted local history and export.
- `internal/ui` - terminal output helpers.

## Key Design Decisions

- JSONL typed protocol replaces plain string parsing.
- Per-client outbound channels prevent blocking writes under room mutex.
- Room token entropy uses `crypto/rand`, not `math/rand`.
- Delivery acknowledgements allow retries and transparent reconnect recovery.
