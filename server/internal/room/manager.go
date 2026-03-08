package room

import "sync"

type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

func (m *Manager) CreateRoom() (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		code, err := NewToken()
		if err != nil {
			return nil, err
		}
		if _, exists := m.rooms[code]; exists {
			continue
		}

		newRoom := New(code)
		m.rooms[code] = newRoom
		return newRoom, nil
	}
}

func (m *Manager) GetRoom(code string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[code]
	return r, ok
}

func (m *Manager) DeleteRoom(code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, code)
}
