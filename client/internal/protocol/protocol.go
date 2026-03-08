package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	Version        = 1
	MaxMessageSize = 1024 * 1024
)

type MessageType string

const (
	TypeHello       MessageType = "hello"
	TypeCreateRoom  MessageType = "create_room"
	TypeJoinRoom    MessageType = "join_room"
	TypeRoomCreated MessageType = "room_created"
	TypeRoomJoined  MessageType = "room_joined"
	TypeChat        MessageType = "chat"
	TypeSystem      MessageType = "system"
	TypeError       MessageType = "error"
	TypeAck         MessageType = "ack"
	TypePing        MessageType = "ping"
	TypePong        MessageType = "pong"
	TypeCommand     MessageType = "command"
	TypeUsers       MessageType = "users"
	TypeLeave       MessageType = "leave"
)

type Envelope struct {
	Version   int         `json:"version"`
	Type      MessageType `json:"type"`
	ID        string      `json:"id,omitempty"`
	Room      string      `json:"room,omitempty"`
	Sender    string      `json:"sender,omitempty"`
	Payload   string      `json:"payload,omitempty"`
	Timestamp time.Time   `json:"ts"`
}

func NewEnvelope(t MessageType) Envelope {
	return Envelope{Version: Version, Type: t, Timestamp: time.Now().UTC()}
}

func (e *Envelope) Validate() error {
	if e.Version != Version {
		return fmt.Errorf("unsupported protocol version: %d", e.Version)
	}
	if e.Type == "" {
		return errors.New("missing type")
	}
	if strings.ContainsRune(e.Payload, '\n') {
		return errors.New("payload contains newline")
	}
	return nil
}

type Decoder struct {
	reader *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{reader: bufio.NewReaderSize(r, 16*1024)}
}

func (d *Decoder) Decode() (Envelope, error) {
	line, err := d.reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && strings.TrimSpace(line) == "" {
			return Envelope{}, io.EOF
		}
		if strings.TrimSpace(line) == "" {
			return Envelope{}, err
		}
	}
	if len(line) > MaxMessageSize {
		return Envelope{}, fmt.Errorf("message exceeds %d bytes", MaxMessageSize)
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return Envelope{}, errors.New("empty message")
	}

	var env Envelope
	if unmarshalErr := json.Unmarshal([]byte(line), &env); unmarshalErr != nil {
		return Envelope{}, fmt.Errorf("decode message: %w", unmarshalErr)
	}
	if validateErr := env.Validate(); validateErr != nil {
		return Envelope{}, validateErr
	}
	return env, nil
}

func Encode(w io.Writer, env Envelope) error {
	if env.Timestamp.IsZero() {
		env.Timestamp = time.Now().UTC()
	}
	if env.Version == 0 {
		env.Version = Version
	}
	if err := env.Validate(); err != nil {
		return err
	}

	b, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}

	if _, err = w.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}
