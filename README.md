# 💬 Terminal Messenger

A simple terminal-based messenger written in Go. Create secure connections between your devices or chat with friends through CLI.

## 📁 Project Structure

```
Messenger/
├── server/
│   ├── server.go   # Main server logic
│   ├── room.go     # Room management (create, join, broadcast)
│   ├── code.go     # Funny 8-digit room code generator
│   └── go.mod
└── client/
    ├── client.go   # Client logic (connect, send, receive)
    └── go.mod
```

## 🚀 Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/TimofeySukh/Messenger.git
cd Messenger
```

### 2. Install Go (if not installed)

```bash
# Ubuntu/Debian
sudo apt install golang

# Fedora
sudo dnf install golang

# macOS
brew install go
```

### 3. Run the server

```bash
cd server
go run .
```

Server will start on port `8080`.

### 4. Run the client

Open a new terminal:

```bash
cd client

# Option 1: Pass IP directly
go run client.go -ip=192.168.1.100:8080

# Option 2: Use environment variable
export SERVER_IP=192.168.1.100:8080
go run client.go
```

## 💡 Usage

1. **Create a room**: One user creates a room and gets an 8-digit code
2. **Share the code**: Tell your friend the code
3. **Connect**: Friend enters the code to join
4. **Chat**: Send messages back and forth

## 🔧 Configuration

| Method | Example |
|--------|---------|
| Flag | `go run client.go -ip=192.168.1.100:8080` |
| Environment | `export SERVER_IP=192.168.1.100:8080` |

Port `8080` is added automatically if not specified.

## 📝 License

MIT
