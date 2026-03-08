package session

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	cryptoutil "messenger-client/internal/crypto"
	"messenger-client/internal/history"
	"messenger-client/internal/protocol"
	"messenger-client/internal/ui"
)

type Mode string

const (
	ModeCreate Mode = "create"
	ModeJoin   Mode = "join"
)

type Options struct {
	ServerAddr        string
	Username          string
	Mode              Mode
	RoomToken         string
	EncryptionKey     string
	ReconnectAttempts int
	RetryInterval     time.Duration
	MaxRetries        int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	DialTimeout       time.Duration
	In                io.Reader
	Out               io.Writer
	Err               io.Writer
}

func DefaultOptions() Options {
	return Options{
		Mode:              ModeCreate,
		ReconnectAttempts: 5,
		RetryInterval:     2 * time.Second,
		MaxRetries:        3,
		ReadTimeout:       90 * time.Second,
		WriteTimeout:      10 * time.Second,
		DialTimeout:       5 * time.Second,
		In:                os.Stdin,
		Out:               os.Stdout,
		Err:               os.Stderr,
	}
}

type Runner struct {
	opts       Options
	roomToken  string
	username   string
	key        string
	inputLines chan string
	pending    map[string]*pendingMessage
	pendingMu  sync.Mutex
}

type pendingMessage struct {
	env     protocol.Envelope
	sentAt  time.Time
	retries int
	onAck   func()
}

type liveConnection struct {
	conn     net.Conn
	decoder  *protocol.Decoder
	outbound chan protocol.Envelope
	incoming chan protocol.Envelope
	readErr  chan error
	writeErr chan error
	done     chan struct{}
	once     sync.Once
}

func NewRunner(opts Options) *Runner {
	defaults := DefaultOptions()

	if opts.ReconnectAttempts <= 0 {
		opts.ReconnectAttempts = defaults.ReconnectAttempts
	}
	if opts.RetryInterval <= 0 {
		opts.RetryInterval = defaults.RetryInterval
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = defaults.MaxRetries
	}
	if opts.ReadTimeout <= 0 {
		opts.ReadTimeout = defaults.ReadTimeout
	}
	if opts.WriteTimeout <= 0 {
		opts.WriteTimeout = defaults.WriteTimeout
	}
	if opts.DialTimeout <= 0 {
		opts.DialTimeout = defaults.DialTimeout
	}
	if opts.In == nil {
		opts.In = defaults.In
	}
	if opts.Out == nil {
		opts.Out = defaults.Out
	}
	if opts.Err == nil {
		opts.Err = defaults.Err
	}
	if opts.Mode == "" {
		opts.Mode = defaults.Mode
	}

	return &Runner{
		opts:       opts,
		roomToken:  strings.TrimSpace(opts.RoomToken),
		username:   strings.TrimSpace(opts.Username),
		key:        strings.TrimSpace(opts.EncryptionKey),
		inputLines: make(chan string, 256),
		pending:    make(map[string]*pendingMessage),
	}
}

func (r *Runner) Run(ctx context.Context) error {
	if r.username == "" {
		return errors.New("username cannot be empty")
	}
	if len(r.username) > 24 {
		r.username = r.username[:24]
	}

	if r.opts.Mode == ModeJoin {
		if r.roomToken == "" {
			return errors.New("room token is required for join mode")
		}
		if !cryptoutil.IsValidKey(r.key) {
			return errors.New("invalid encryption key")
		}
	} else {
		if r.key == "" {
			generated, err := cryptoutil.GenerateEncryptionKey()
			if err != nil {
				return fmt.Errorf("generate encryption key: %w", err)
			}
			r.key = generated
		}
	}

	conn, err := r.connect(ctx, r.opts.Mode)
	if err != nil {
		return err
	}
	defer conn.Close()

	r.printBanner()
	r.printSessionInfo()

	go r.readInput()

	retryTicker := time.NewTicker(r.opts.RetryInterval)
	defer retryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.sendLeave(conn)
			return nil
		case line, ok := <-r.inputLines:
			if !ok {
				r.sendLeave(conn)
				return nil
			}
			exit, inputErr := r.handleInput(line, conn)
			if inputErr != nil {
				fmt.Fprintf(r.opts.Err, "error: %v\n", inputErr)
			}
			if exit {
				r.sendLeave(conn)
				return nil
			}
		case env := <-conn.incoming:
			r.handleIncoming(env, conn)
		case readErr := <-conn.readErr:
			if recErr := r.reconnect(ctx, &conn, readErr); recErr != nil {
				return recErr
			}
		case writeErr := <-conn.writeErr:
			if recErr := r.reconnect(ctx, &conn, writeErr); recErr != nil {
				return recErr
			}
		case <-retryTicker.C:
			r.resendPending(conn)
		}
	}
}

func (r *Runner) reconnect(ctx context.Context, current **liveConnection, reason error) error {
	fmt.Fprintf(r.opts.Err, "connection lost: %v\n", reason)
	(*current).Close()

	for attempt := 1; attempt <= r.opts.ReconnectAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		delay := time.Duration(attempt) * time.Second
		fmt.Fprintf(r.opts.Err, "reconnecting attempt %d/%d in %s...\n", attempt, r.opts.ReconnectAttempts, delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		conn, err := r.connect(ctx, ModeJoin)
		if err != nil {
			fmt.Fprintf(r.opts.Err, "reconnect attempt %d failed: %v\n", attempt, err)
			continue
		}

		fmt.Fprintln(r.opts.Out, "reconnected")
		*current = conn
		r.resendPending(conn)
		return nil
	}

	return errors.New("failed to reconnect")
}

func (r *Runner) connect(ctx context.Context, mode Mode) (*liveConnection, error) {
	dialer := net.Dialer{Timeout: r.opts.DialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", normalizeAddress(r.opts.ServerAddr))
	if err != nil {
		return nil, fmt.Errorf("dial server: %w", err)
	}

	decoder := protocol.NewDecoder(conn)

	if err := conn.SetWriteDeadline(time.Now().Add(r.opts.WriteTimeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	hello := protocol.NewEnvelope(protocol.TypeHello)
	hello.Sender = r.username
	if err := protocol.Encode(conn, hello); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("send hello: %w", err)
	}

	request := protocol.NewEnvelope(protocol.TypeCreateRoom)
	if mode == ModeJoin || r.roomToken != "" {
		request.Type = protocol.TypeJoinRoom
		request.Room = r.roomToken
	}
	if err := protocol.Encode(conn, request); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("send room request: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(r.opts.ReadTimeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	response, err := decoder.Decode()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("read handshake response: %w", err)
	}

	switch response.Type {
	case protocol.TypeRoomCreated:
		r.roomToken = response.Room
	case protocol.TypeRoomJoined:
		if response.Room != "" {
			r.roomToken = response.Room
		}
	case protocol.TypeError:
		_ = conn.Close()
		return nil, fmt.Errorf("server rejected connection: %s", response.Payload)
	default:
		_ = conn.Close()
		return nil, fmt.Errorf("unexpected handshake response: %s", response.Type)
	}

	live := &liveConnection{
		conn:     conn,
		decoder:  decoder,
		outbound: make(chan protocol.Envelope, 256),
		incoming: make(chan protocol.Envelope, 256),
		readErr:  make(chan error, 1),
		writeErr: make(chan error, 1),
		done:     make(chan struct{}),
	}
	go r.readLoop(live)
	go r.writeLoop(live)

	if r.opts.Mode == ModeCreate {
		r.opts.Mode = ModeJoin
	}
	return live, nil
}

func (r *Runner) readLoop(live *liveConnection) {
	for {
		if err := live.conn.SetReadDeadline(time.Now().Add(r.opts.ReadTimeout)); err != nil {
			live.readErr <- err
			return
		}
		env, err := live.decoder.Decode()
		if err != nil {
			select {
			case <-live.done:
				return
			default:
				live.readErr <- err
				return
			}
		}
		select {
		case <-live.done:
			return
		case live.incoming <- env:
		}
	}
}

func (r *Runner) writeLoop(live *liveConnection) {
	for {
		select {
		case <-live.done:
			return
		case env := <-live.outbound:
			if err := live.conn.SetWriteDeadline(time.Now().Add(r.opts.WriteTimeout)); err != nil {
				live.writeErr <- err
				return
			}
			if err := protocol.Encode(live.conn, env); err != nil {
				select {
				case <-live.done:
					return
				default:
					live.writeErr <- err
					return
				}
			}
		}
	}
}

func (c *liveConnection) Close() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (r *Runner) enqueue(live *liveConnection, env protocol.Envelope) {
	select {
	case live.outbound <- env:
	default:
		fmt.Fprintln(r.opts.Err, "outbound queue is full, dropping message")
	}
}

func (r *Runner) readInput() {
	scanner := bufio.NewScanner(r.opts.In)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		r.inputLines <- line
	}
	close(r.inputLines)
}

func (r *Runner) handleInput(line string, live *liveConnection) (bool, error) {
	if line == "" {
		return false, nil
	}

	if strings.HasPrefix(line, "/") {
		return r.handleCommand(line, live)
	}

	id, err := newMessageID()
	if err != nil {
		return false, err
	}

	enc, err := cryptoutil.Encrypt(line, r.key)
	if err != nil {
		return false, err
	}

	env := protocol.NewEnvelope(protocol.TypeChat)
	env.ID = id
	env.Room = r.roomToken
	env.Sender = r.username
	env.Payload = enc

	r.pendingMu.Lock()
	r.pending[id] = &pendingMessage{env: env, sentAt: time.Now()}
	r.pendingMu.Unlock()

	r.enqueue(live, env)
	_ = history.Append(r.roomToken, r.key, r.username, line)

	return false, nil
}

func (r *Runner) handleCommand(line string, live *liveConnection) (bool, error) {
	parts := strings.Fields(line)
	cmd := strings.ToLower(strings.TrimPrefix(parts[0], "/"))

	switch cmd {
	case "help":
		fmt.Fprintln(r.opts.Out, "Commands: /help /users /nick <name> /leave /rekey /history [n] /export <path>")
		return false, nil
	case "users":
		env := protocol.NewEnvelope(protocol.TypeCommand)
		env.Room = r.roomToken
		env.Sender = r.username
		env.Payload = "users"
		r.enqueue(live, env)
		return false, nil
	case "nick":
		if len(parts) < 2 {
			return false, errors.New("usage: /nick <name>")
		}
		newName := strings.TrimSpace(parts[1])
		if newName == "" {
			return false, errors.New("nickname cannot be empty")
		}
		env := protocol.NewEnvelope(protocol.TypeCommand)
		env.Room = r.roomToken
		env.Sender = r.username
		env.Payload = "nick " + newName
		r.username = newName
		r.enqueue(live, env)
		return false, nil
	case "leave":
		return true, nil
	case "rekey":
		newKey, err := cryptoutil.GenerateEncryptionKey()
		if err != nil {
			return false, err
		}

		id, err := newMessageID()
		if err != nil {
			return false, err
		}

		controlPayload := "__REKEY__:" + newKey
		enc, err := cryptoutil.Encrypt(controlPayload, r.key)
		if err != nil {
			return false, err
		}

		env := protocol.NewEnvelope(protocol.TypeChat)
		env.ID = id
		env.Room = r.roomToken
		env.Sender = r.username
		env.Payload = enc

		r.pendingMu.Lock()
		r.pending[id] = &pendingMessage{
			env:    env,
			sentAt: time.Now(),
			onAck: func() {
				r.key = newKey
				fmt.Fprintln(r.opts.Out, "encryption key rotated. Share the new key securely:")
				fmt.Fprintln(r.opts.Out, newKey)
			},
		}
		r.pendingMu.Unlock()

		r.enqueue(live, env)
		return false, nil
	case "history":
		count := 20
		if len(parts) >= 2 {
			if parsed, err := strconv.Atoi(parts[1]); err == nil && parsed > 0 {
				count = parsed
			}
		}
		entries, err := history.Last(r.roomToken, r.key, count)
		if err != nil {
			return false, err
		}
		for _, entry := range entries {
			fmt.Fprintf(r.opts.Out, "%s [%s] %s\n", entry.Timestamp.Format("15:04:05"), entry.Sender, entry.Message)
		}
		return false, nil
	case "export":
		if len(parts) < 2 {
			return false, errors.New("usage: /export <path>")
		}
		path := parts[1]
		if strings.HasPrefix(path, "~/") {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
		if err := history.Export(r.roomToken, r.key, path); err != nil {
			return false, err
		}
		fmt.Fprintf(r.opts.Out, "history exported to %s\n", path)
		return false, nil
	default:
		return false, fmt.Errorf("unknown command: %s", cmd)
	}
}

func (r *Runner) handleIncoming(env protocol.Envelope, live *liveConnection) {
	switch env.Type {
	case protocol.TypeChat:
		message, err := cryptoutil.Decrypt(env.Payload, r.key)
		if err != nil {
			fmt.Fprintf(r.opts.Out, "[%s] [UNREADABLE MESSAGE]\n", env.Sender)
			return
		}
		if strings.HasPrefix(message, "__REKEY__:") {
			if env.Sender != r.username {
				candidate := strings.TrimPrefix(message, "__REKEY__:")
				if cryptoutil.IsValidKey(candidate) {
					r.key = candidate
					fmt.Fprintf(r.opts.Out, "[system] encryption key updated by %s\n", env.Sender)
				}
			}
			return
		}
		if strings.Contains(message, "@"+r.username) && env.Sender != r.username {
			fmt.Fprint(r.opts.Out, "\a")
		}
		fmt.Fprintf(r.opts.Out, "[%s] %s\n", env.Sender, message)
		_ = history.Append(r.roomToken, r.key, env.Sender, message)
	case protocol.TypeSystem:
		fmt.Fprintf(r.opts.Out, "[system] %s\n", env.Payload)
	case protocol.TypeUsers:
		fmt.Fprintf(r.opts.Out, "[users] %s\n", env.Payload)
	case protocol.TypeError:
		fmt.Fprintf(r.opts.Err, "server error: %s\n", env.Payload)
	case protocol.TypePing:
		pong := protocol.NewEnvelope(protocol.TypePong)
		pong.Room = r.roomToken
		pong.Sender = r.username
		r.enqueue(live, pong)
	case protocol.TypeAck:
		r.handleAck(env.ID)
	}
}

func (r *Runner) handleAck(id string) {
	r.pendingMu.Lock()
	pending, ok := r.pending[id]
	if ok {
		delete(r.pending, id)
	}
	r.pendingMu.Unlock()

	if ok && pending.onAck != nil {
		pending.onAck()
	}
}

func (r *Runner) resendPending(live *liveConnection) {
	now := time.Now()
	retry := make([]protocol.Envelope, 0)
	failed := make([]string, 0)

	r.pendingMu.Lock()
	for id, pending := range r.pending {
		if now.Sub(pending.sentAt) < r.opts.RetryInterval {
			continue
		}
		if pending.retries >= r.opts.MaxRetries {
			failed = append(failed, id)
			continue
		}
		pending.retries++
		pending.sentAt = now
		retry = append(retry, pending.env)
	}
	for _, id := range failed {
		delete(r.pending, id)
	}
	r.pendingMu.Unlock()

	for _, env := range retry {
		r.enqueue(live, env)
	}
	for _, id := range failed {
		fmt.Fprintf(r.opts.Err, "message %s failed delivery after retries\n", id)
	}
}

func (r *Runner) sendLeave(live *liveConnection) {
	env := protocol.NewEnvelope(protocol.TypeLeave)
	env.Room = r.roomToken
	env.Sender = r.username
	r.enqueue(live, env)
}

func (r *Runner) printBanner() {
	ui.PrintBanner(r.opts.Out)
}

func (r *Runner) printSessionInfo() {
	ui.PrintSessionInfo(r.opts.Out, r.username, r.roomToken, r.key)
}

func normalizeAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "127.0.0.1:8080"
	}
	if !strings.Contains(addr, ":") {
		return addr + ":8080"
	}
	return addr
}

func newMessageID() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), "="), nil
}

func (r *Runner) RoomToken() string {
	return r.roomToken
}

func (r *Runner) EncryptionKey() string {
	return r.key
}

func (r *Runner) Username() string {
	return r.username
}
