package transport

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"messenger-server/internal/protocol"
	"messenger-server/internal/room"
)

type Config struct {
	ListenAddr    string
	JoinLimit     int
	JoinWindow    time.Duration
	OutboundQueue int
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	PingInterval  time.Duration
}

func DefaultConfig() Config {
	return Config{
		ListenAddr:    "0.0.0.0:8080",
		JoinLimit:     30,
		JoinWindow:    time.Minute,
		OutboundQueue: 128,
		ReadTimeout:   90 * time.Second,
		WriteTimeout:  10 * time.Second,
		PingInterval:  25 * time.Second,
	}
}

type Server struct {
	cfg     Config
	rooms   *room.Manager
	limiter *room.JoinLimiter
	logger  *log.Logger
}

func New(cfg Config, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}
	return &Server{
		cfg:     cfg,
		rooms:   room.NewManager(),
		limiter: room.NewJoinLimiter(cfg.JoinLimit, cfg.JoinWindow),
		logger:  logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	s.logger.Printf("server listening on %s", s.cfg.ListenAddr)

	var wg sync.WaitGroup
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				break
			}
			s.logger.Printf("accept error: %v", err)
			continue
		}

		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			s.handleConnection(ctx, c)
		}(conn)
	}

	wg.Wait()
	return nil
}

var sessionCounter uint64

type clientSession struct {
	id       string
	conn     net.Conn
	decoder  *protocol.Decoder
	outbound chan protocol.Envelope
	done     chan struct{}
	once     sync.Once

	username string
	room     *room.Room
	roomCode string
	seen     map[string]time.Time
}

func newSession(conn net.Conn, queueSize int) *clientSession {
	id := atomic.AddUint64(&sessionCounter, 1)
	return &clientSession{
		id:       fmt.Sprintf("client-%d", id),
		conn:     conn,
		decoder:  protocol.NewDecoder(conn),
		outbound: make(chan protocol.Envelope, queueSize),
		done:     make(chan struct{}),
		seen:     make(map[string]time.Time),
	}
}

func (c *clientSession) Close() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *clientSession) Send(msg protocol.Envelope) bool {
	select {
	case c.outbound <- msg:
		return true
	default:
		return false
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	session := newSession(conn, s.cfg.OutboundQueue)
	defer session.Close()

	if err := s.handleHandshake(session); err != nil {
		s.writeDirect(session.conn, protocol.NewEnvelope(protocol.TypeError), err.Error())
		s.logger.Printf("handshake failed (%s): %v", conn.RemoteAddr().String(), err)
		return
	}

	writerErr := make(chan error, 1)
	go func() {
		writerErr <- s.writerLoop(session)
	}()

	pingStop := make(chan struct{})
	go s.pingLoop(session, pingStop)
	defer close(pingStop)

	joinNotice := protocol.NewEnvelope(protocol.TypeSystem)
	joinNotice.Room = session.roomCode
	joinNotice.Sender = "system"
	joinNotice.Payload = fmt.Sprintf("%s joined the room", session.username)
	session.room.Broadcast(joinNotice, session.id)

	if runErr := s.readLoop(ctx, session); runErr != nil && !errors.Is(runErr, net.ErrClosed) {
		s.logger.Printf("client %s disconnected with error: %v", session.username, runErr)
	}

	leaveNotice := protocol.NewEnvelope(protocol.TypeSystem)
	leaveNotice.Room = session.roomCode
	leaveNotice.Sender = "system"
	leaveNotice.Payload = fmt.Sprintf("%s left the room", session.username)

	session.room.Remove(session.id)
	session.room.Broadcast(leaveNotice, session.id)
	if session.room.Count() == 0 {
		s.rooms.DeleteRoom(session.roomCode)
	}

	session.Close()
	select {
	case <-writerErr:
	case <-time.After(2 * time.Second):
	}
}

func (s *Server) handleHandshake(session *clientSession) error {
	if err := session.conn.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout)); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}

	hello, err := session.decoder.Decode()
	if err != nil {
		return fmt.Errorf("read hello: %w", err)
	}
	if hello.Type != protocol.TypeHello {
		return errors.New("expected hello message")
	}

	username := normalizeUsername(hello.Sender)
	if username == "" {
		return errors.New("username is required")
	}
	session.username = username

	request, err := session.decoder.Decode()
	if err != nil {
		return fmt.Errorf("read room request: %w", err)
	}

	switch request.Type {
	case protocol.TypeCreateRoom:
		selectedRoom, createErr := s.rooms.CreateRoom()
		if createErr != nil {
			return createErr
		}
		session.room = selectedRoom
		session.roomCode = selectedRoom.Code

		member := &room.Member{ID: session.id, Username: session.username, Outbound: session.outbound, Disconnect: session.Close}
		selectedRoom.Add(member)

		created := protocol.NewEnvelope(protocol.TypeRoomCreated)
		created.Room = selectedRoom.Code
		created.Sender = "system"
		created.Payload = "room created"
		return protocol.Encode(session.conn, created)
	case protocol.TypeJoinRoom:
		roomCode := strings.TrimSpace(request.Room)
		if roomCode == "" {
			return errors.New("room token is required")
		}
		remoteHost := session.conn.RemoteAddr().String()
		if host, _, splitErr := net.SplitHostPort(remoteHost); splitErr == nil {
			remoteHost = host
		}
		if !s.limiter.Allow(remoteHost) {
			return errors.New("too many join attempts, try again later")
		}

		selectedRoom, ok := s.rooms.GetRoom(roomCode)
		if !ok {
			return errors.New("room not found")
		}
		session.room = selectedRoom
		session.roomCode = roomCode

		member := &room.Member{ID: session.id, Username: session.username, Outbound: session.outbound, Disconnect: session.Close}
		selectedRoom.Add(member)

		joined := protocol.NewEnvelope(protocol.TypeRoomJoined)
		joined.Room = selectedRoom.Code
		joined.Sender = "system"
		joined.Payload = "connected"
		return protocol.Encode(session.conn, joined)
	default:
		return errors.New("unknown room command")
	}
}

func (s *Server) readLoop(ctx context.Context, session *clientSession) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := session.conn.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout)); err != nil {
			return err
		}

		env, err := session.decoder.Decode()
		if err != nil {
			return err
		}

		s.cleanupSeen(session)

		switch env.Type {
		case protocol.TypeChat:
			if env.ID == "" {
				continue
			}
			if _, exists := session.seen[env.ID]; exists {
				ack := protocol.NewEnvelope(protocol.TypeAck)
				ack.ID = env.ID
				ack.Room = session.roomCode
				ack.Sender = "system"
				session.Send(ack)
				continue
			}
			session.seen[env.ID] = time.Now()

			ack := protocol.NewEnvelope(protocol.TypeAck)
			ack.ID = env.ID
			ack.Room = session.roomCode
			ack.Sender = "system"
			session.Send(ack)

			forward := protocol.NewEnvelope(protocol.TypeChat)
			forward.ID = env.ID
			forward.Room = session.roomCode
			forward.Sender = session.username
			forward.Payload = env.Payload
			session.room.Broadcast(forward, session.id)
		case protocol.TypeCommand:
			s.handleCommand(session, env.Payload)
		case protocol.TypePing:
			pong := protocol.NewEnvelope(protocol.TypePong)
			pong.Room = session.roomCode
			pong.Sender = "system"
			session.Send(pong)
		case protocol.TypePong:
			continue
		case protocol.TypeLeave:
			return nil
		default:
			errMsg := protocol.NewEnvelope(protocol.TypeError)
			errMsg.Sender = "system"
			errMsg.Payload = "unsupported message type"
			session.Send(errMsg)
		}
	}
}

func (s *Server) writerLoop(session *clientSession) error {
	for {
		select {
		case <-session.done:
			return nil
		case msg := <-session.outbound:
			if err := session.conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout)); err != nil {
				return err
			}
			if err := protocol.Encode(session.conn, msg); err != nil {
				return err
			}
		}
	}
}

func (s *Server) pingLoop(session *clientSession, stop <-chan struct{}) {
	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-session.done:
			return
		case <-ticker.C:
			ping := protocol.NewEnvelope(protocol.TypePing)
			ping.Room = session.roomCode
			ping.Sender = "system"
			session.Send(ping)
		}
	}
}

func (s *Server) writeDirect(conn net.Conn, base protocol.Envelope, payload string) {
	base.Payload = payload
	_ = conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout))
	_ = protocol.Encode(conn, base)
}

func (s *Server) handleCommand(session *clientSession, payload string) {
	parts := strings.Fields(payload)
	if len(parts) == 0 {
		return
	}

	switch strings.ToLower(parts[0]) {
	case "users":
		users := protocol.NewEnvelope(protocol.TypeUsers)
		users.Room = session.roomCode
		users.Sender = "system"
		users.Payload = strings.Join(session.room.Users(), ", ")
		session.Send(users)
	case "nick":
		if len(parts) < 2 {
			msg := protocol.NewEnvelope(protocol.TypeError)
			msg.Sender = "system"
			msg.Payload = "usage: /nick <name>"
			session.Send(msg)
			return
		}
		newName := normalizeUsername(parts[1])
		if newName == "" {
			msg := protocol.NewEnvelope(protocol.TypeError)
			msg.Sender = "system"
			msg.Payload = "invalid nickname"
			session.Send(msg)
			return
		}
		oldName := session.username
		session.username = newName
		session.room.Rename(session.id, newName)

		notice := protocol.NewEnvelope(protocol.TypeSystem)
		notice.Room = session.roomCode
		notice.Sender = "system"
		notice.Payload = fmt.Sprintf("%s is now known as %s", oldName, newName)
		session.room.Broadcast(notice, "")
	default:
		msg := protocol.NewEnvelope(protocol.TypeError)
		msg.Sender = "system"
		msg.Payload = "unknown command"
		session.Send(msg)
	}
}

func (s *Server) cleanupSeen(session *clientSession) {
	if len(session.seen) < 4096 {
		return
	}
	cutoff := time.Now().Add(-30 * time.Minute)
	for id, t := range session.seen {
		if t.Before(cutoff) {
			delete(session.seen, id)
		}
	}
}

func normalizeUsername(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 24 {
		name = name[:24]
	}
	return name
}
