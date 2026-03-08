package room

import (
	"sort"
	"sync"

	"messenger-server/internal/protocol"
)

type Member struct {
	ID         string
	Username   string
	Outbound   chan protocol.Envelope
	Disconnect func()
}

type Room struct {
	Code    string
	mu      sync.RWMutex
	members map[string]*Member
}

func New(code string) *Room {
	return &Room{
		Code:    code,
		members: make(map[string]*Member),
	}
}

func (r *Room) Add(member *Member) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[member.ID] = member
}

func (r *Room) Remove(memberID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.members, memberID)
}

func (r *Room) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members)
}

func (r *Room) Users() []string {
	r.mu.RLock()
	users := make([]string, 0, len(r.members))
	for _, member := range r.members {
		users = append(users, member.Username)
	}
	r.mu.RUnlock()

	sort.Strings(users)
	return users
}

func (r *Room) Rename(memberID, username string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	member, ok := r.members[memberID]
	if !ok {
		return false
	}
	member.Username = username
	return true
}

func (r *Room) Broadcast(msg protocol.Envelope, excludeMemberID string) {
	r.mu.RLock()
	targets := make([]*Member, 0, len(r.members))
	for id, member := range r.members {
		if id == excludeMemberID {
			continue
		}
		targets = append(targets, member)
	}
	r.mu.RUnlock()

	for _, member := range targets {
		select {
		case member.Outbound <- msg:
		default:
			if member.Disconnect != nil {
				go member.Disconnect()
			}
		}
	}
}
