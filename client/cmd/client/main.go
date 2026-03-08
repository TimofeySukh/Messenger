package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"messenger-client/internal/config"
	"messenger-client/internal/crypto"
	"messenger-client/internal/session"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	modeFlag := flag.String("mode", "", "Mode: create or join")
	ipFlag := flag.String("ip", "", "Server address (host:port)")
	userFlag := flag.String("username", "", "Username")
	roomFlag := flag.String("room", "", "Room token or alias")
	keyFlag := flag.String("key", "", "Room encryption key")
	aliasFlag := flag.String("alias", "", "Alias to save for the room after session starts")
	flag.Parse()

	mode := session.Mode(strings.ToLower(strings.TrimSpace(*modeFlag)))
	if mode == "" {
		mode = session.ModeCreate
	}
	if mode != session.ModeCreate && mode != session.ModeJoin {
		fmt.Fprintln(os.Stderr, "mode must be 'create' or 'join'")
		os.Exit(1)
	}

	serverAddr := strings.TrimSpace(*ipFlag)
	if serverAddr == "" {
		serverAddr = cfg.Server
	}
	if serverAddr == "" {
		serverAddr = "127.0.0.1:8080"
	}

	username := strings.TrimSpace(*userFlag)
	if username == "" {
		username = strings.TrimSpace(cfg.Username)
	}

	room := strings.TrimSpace(*roomFlag)
	if aliasRoom, ok := cfg.Aliases[room]; ok {
		room = aliasRoom
	}

	key := strings.TrimSpace(*keyFlag)

	reader := bufio.NewReader(os.Stdin)
	if username == "" {
		username = prompt(reader, "Username: ")
	}

	if mode == session.ModeJoin {
		if room == "" {
			room = prompt(reader, "Room token: ")
		}
		if key == "" {
			key = prompt(reader, "Encryption key: ")
		}
		if !crypto.IsValidKey(key) {
			fmt.Fprintln(os.Stderr, "invalid encryption key format")
			os.Exit(1)
		}
	}

	opts := session.DefaultOptions()
	opts.ServerAddr = serverAddr
	opts.Username = username
	opts.Mode = mode
	opts.RoomToken = room
	opts.EncryptionKey = key

	runner := session.NewRunner(opts)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runner.Run(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "session error: %v\n", err)
		os.Exit(1)
	}

	cfg.Server = serverAddr
	cfg.Username = runner.Username()
	alias := strings.TrimSpace(*aliasFlag)
	if alias != "" && runner.RoomToken() != "" {
		cfg.Aliases[alias] = runner.RoomToken()
	}

	if saveErr := config.Save(cfg); saveErr != nil {
		fmt.Fprintf(os.Stderr, "failed to save config: %v\n", saveErr)
	}
}

func prompt(reader *bufio.Reader, label string) string {
	for {
		fmt.Print(label)
		value, err := reader.ReadString('\n')
		if err != nil {
			return ""
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
}
