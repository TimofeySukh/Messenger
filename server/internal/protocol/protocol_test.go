package protocol

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	var buf bytes.Buffer

	original := NewEnvelope(TypeChat)
	original.ID = "msg-1"
	original.Room = "ABC123"
	original.Sender = "alice"
	original.Payload = "ciphertext"

	if err := Encode(&buf, original); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := NewDecoder(&buf).Decode()
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Type != original.Type || decoded.ID != original.ID || decoded.Room != original.Room || decoded.Sender != original.Sender || decoded.Payload != original.Payload {
		t.Fatalf("decoded message mismatch: got %+v want %+v", decoded, original)
	}
}

func TestDecodeRejectsInvalidVersion(t *testing.T) {
	input := bytes.NewBufferString(`{"version":999,"type":"chat","ts":"2026-01-01T00:00:00Z"}` + "\n")
	_, err := NewDecoder(input).Decode()
	if err == nil {
		t.Fatal("expected error for invalid protocol version")
	}
}
