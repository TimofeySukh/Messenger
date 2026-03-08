package session

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	cryptoutil "messenger-client/internal/crypto"
	"messenger-client/internal/protocol"
)

func TestLeaveCommand(t *testing.T) {
	r := NewRunner(DefaultOptions())
	live := &liveConnection{outbound: make(chan protocol.Envelope, 1)}
	exit, err := r.handleCommand("/leave", live)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exit {
		t.Fatal("expected /leave to request exit")
	}
}

func TestReconnectAfterDisconnect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan struct{})
	go runMockServer(t, ln, serverDone)
	defer close(serverDone)

	key, err := cryptoutil.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	opts := DefaultOptions()
	opts.ServerAddr = ln.Addr().String()
	opts.Username = "alice"
	opts.Mode = ModeCreate
	opts.EncryptionKey = key
	opts.In = strings.NewReader("")
	opts.Out = io.Discard
	opts.Err = io.Discard
	opts.ReconnectAttempts = 2

	r := NewRunner(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	live, err := r.connect(ctx, ModeCreate)
	if err != nil {
		t.Fatalf("initial connect failed: %v", err)
	}
	defer live.Close()

	if err := r.reconnect(ctx, &live, io.EOF); err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}
}

func runMockServer(t *testing.T, ln net.Listener, done <-chan struct{}) {
	t.Helper()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-done:
				return
			default:
				return
			}
		}

		go func(c net.Conn) {
			defer c.Close()
			dec := protocol.NewDecoder(c)

			hello, err := dec.Decode()
			if err != nil || hello.Type != protocol.TypeHello {
				return
			}
			req, err := dec.Decode()
			if err != nil {
				return
			}

			if req.Type == protocol.TypeCreateRoom {
				resp := protocol.NewEnvelope(protocol.TypeRoomCreated)
				resp.Room = "ROOMTOKEN1"
				_ = protocol.Encode(c, resp)
				return
			}

			resp := protocol.NewEnvelope(protocol.TypeRoomJoined)
			resp.Room = "ROOMTOKEN1"
			_ = protocol.Encode(c, resp)
			select {
			case <-done:
			case <-time.After(200 * time.Millisecond):
			}
		}(conn)
	}
}
