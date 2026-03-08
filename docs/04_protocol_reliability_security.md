# Protocol, Reliability, and Security

## Envelope Contract

Each frame is one JSON object per line:

```json
{
  "version": 1,
  "type": "chat",
  "id": "MSGID",
  "room": "ROOMTOKEN",
  "sender": "alice",
  "payload": "...",
  "ts": "2026-03-08T12:34:56Z"
}
```

## Message Types

- Handshake: `hello`, `create_room`, `join_room`, `room_created`, `room_joined`
- Chat and control: `chat`, `command`, `users`, `leave`
- Reliability and liveness: `ack`, `ping`, `pong`
- Server notifications: `system`, `error`

## Reliability Path

- Client tags each chat with a unique `id`.
- Server sends `ack` for accepted chat IDs.
- Client retries unacked IDs on interval with max retry cap.
- On reconnect, pending queue is resent.

## Concurrency and Backpressure

- Room fanout writes to in-memory per-client queue.
- Full queue marks a slow client and disconnects it.
- Server no longer blocks all room traffic on a single slow socket.

## Security Model

- Message payload encryption/decryption occurs client-side.
- Server routes ciphertext and cannot decrypt payload by design.
- Room token generation uses `crypto/rand`.
- Join limiter reduces token probing velocity.

## Current Limits

- Key distribution remains user-mediated (or via `/rekey` encrypted control message).
- No forward secrecy yet (single room key unless rotated).
- No identity signatures for sender authentication.
