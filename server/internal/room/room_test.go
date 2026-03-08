package room

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"messenger-server/internal/protocol"
)

func TestBroadcastDropsSlowClient(t *testing.T) {
	r := New("ROOM1")
	disconnected := make(chan struct{}, 1)

	slow := &Member{
		ID:       "slow",
		Username: "slow",
		Outbound: make(chan protocol.Envelope, 1),
		Disconnect: func() {
			select {
			case disconnected <- struct{}{}:
			default:
			}
		},
	}
	fast := &Member{ID: "fast", Username: "fast", Outbound: make(chan protocol.Envelope, 1)}

	r.Add(slow)
	r.Add(fast)

	slow.Outbound <- protocol.NewEnvelope(protocol.TypePing)

	msg := protocol.NewEnvelope(protocol.TypeSystem)
	msg.Payload = "hello"
	r.Broadcast(msg, "")

	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("expected disconnect callback for slow client")
	}
}

func TestRoomConcurrentJoinLeave(t *testing.T) {
	r := New("ROOM2")
	const members = 200

	var wg sync.WaitGroup
	for i := 0; i < members; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("id-%d", i)
			r.Add(&Member{ID: id, Username: id, Outbound: make(chan protocol.Envelope, 1)})
		}(i)
	}
	wg.Wait()

	if got := r.Count(); got != members {
		t.Fatalf("unexpected member count after join: got %d want %d", got, members)
	}

	for i := 0; i < members; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("id-%d", i)
			r.Remove(id)
		}(i)
	}
	wg.Wait()

	if got := r.Count(); got != 0 {
		t.Fatalf("unexpected member count after leave: got %d want 0", got)
	}
}
