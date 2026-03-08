package protocol

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	in := NewEnvelope(TypeChat)
	in.ID = "m1"
	in.Room = "ROOM"
	in.Sender = "bob"
	in.Payload = "payload"

	if err := Encode(&buf, in); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	out, err := NewDecoder(&buf).Decode()
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if out.ID != in.ID || out.Room != in.Room || out.Sender != in.Sender || out.Payload != in.Payload {
		t.Fatalf("unexpected decode result: got %+v want %+v", out, in)
	}
}
