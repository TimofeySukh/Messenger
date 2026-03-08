# Server File Map

## `cmd/server/main.go`

- Builds server config.
- Instantiates transport server.
- Handles `SIGINT` and `SIGTERM` for graceful stop.

## `internal/protocol/protocol.go`

- Defines message envelope fields and message types.
- Validates protocol version and basic frame constraints.
- Encodes and decodes newline-delimited JSON frames.

## `internal/protocol/protocol_test.go`

- Round-trip encode/decode test.
- Invalid protocol version rejection test.

## `internal/room/token.go`

- Generates 10-char room token from cryptographically secure random bytes.

## `internal/room/limiter.go`

- Sliding-window join limiter keyed by remote host.
- Protects room join endpoint from high-frequency probing.

## `internal/room/room.go`

- Maintains room membership map.
- Broadcasts to member outbound queues.
- Disconnect callback for slow clients whose queue is full.

## `internal/room/manager.go`

- Creates, retrieves, and deletes rooms.
- Enforces token uniqueness.

## `internal/room/room_test.go`

- Verifies slow-client disconnect behavior.
- Verifies concurrent join/leave correctness.

## `internal/transport/server.go`

- Accept loop and connection lifecycle.
- Handshake (`hello` + `create_room/join_room`).
- Reader/writer goroutines per connection.
- Command handling (`users`, `nick`), ping/pong, dedupe + ack.
- Cleanup on disconnect and empty-room removal.
