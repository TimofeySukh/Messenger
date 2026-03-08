# Terminal Messenger (CLI, End-to-End Encrypted)

Terminal Messenger is a TCP CLI chat with end-to-end message encryption (AES-256-GCM), typed JSONL protocol, reconnect support, delivery acknowledgements, and encrypted local history.

## Highlights

- Strict typed protocol (`version`, `type`, `id`, `room`, `sender`, `payload`, `ts`)
- Room tokens generated with `crypto/rand` + base32
- Per-client outbound queues on server (no socket writes under room lock)
- Client retry + ack flow for more reliable delivery
- Graceful shutdown and reconnect attempts
- Slash commands: `/help`, `/users`, `/nick`, `/leave`, `/rekey`, `/history`, `/export`
- Encrypted local history in `~/.messenger/history`
- Client config in `~/.messenger/config.toml` (server, username, aliases)

## Project Structure

```text
Messenger/
├── server/
│   ├── cmd/server/main.go
│   ├── internal/protocol/
│   ├── internal/room/
│   └── internal/transport/
├── client/
│   ├── cmd/client/main.go
│   ├── internal/config/
│   ├── internal/crypto/
│   ├── internal/history/
│   ├── internal/protocol/
│   ├── internal/session/
│   └── internal/ui/
├── docs/
└── AGENTS.md
```

## Run

### Server

```bash
cd server
go run ./cmd/server
```

### Client

```bash
cd client
# create room
go run ./cmd/client -ip=127.0.0.1:8080 -mode=create -username=alice

# join room
go run ./cmd/client -ip=127.0.0.1:8080 -mode=join -username=bob -room=<ROOM_TOKEN> -key=<ROOM_KEY>
```

## CLI Commands

- `/help` show command list
- `/users` request users in current room
- `/nick <name>` update nickname
- `/leave` leave room and exit
- `/rekey` rotate room key (distributed through encrypted control message)
- `/history [n]` show last `n` locally stored messages (default 20)
- `/export <path>` export decrypted local history to a text file

## Tests

```bash
cd server && go test ./...
cd client && go test ./...
```

## Documentation

Detailed architecture and file-level breakdown is in [`docs/`](./docs).
