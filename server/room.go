package main

import (
	"net"
	"sync"
)

// ============================================================
// ЛОГИКА КОМНАТ
// ============================================================

// Client — один подключённый пользователь
type Client struct {
	Conn     net.Conn // соединение с клиентом
	Username string   // имя пользователя
}

// Room — комната чата
type Room struct {
	Code    string     // 5-значный код комнаты
	Clients []*Client  // список клиентов в комнате
	mu      sync.Mutex // мьютекс для безопасного доступа из разных горутин
}

// ============================================================
// ХРАНИЛИЩЕ КОМНАТ
// ============================================================

// rooms — все активные комнаты
// Ключ: код комнаты (например "12345")
// Значение: указатель на комнату
var rooms = make(map[string]*Room)

// roomsMu — мьютекс для безопасного доступа к rooms
var roomsMu sync.Mutex

// ============================================================
// ФУНКЦИИ ДЛЯ РАБОТЫ С КОМНАТАМИ
// ============================================================

// CreateRoom создаёт новую комнату и возвращает её код
func CreateRoom() string {
	roomsMu.Lock()         // блокируем доступ другим горутинам
	defer roomsMu.Unlock() // разблокируем когда функция закончится

	// Генерируем код (используем твою функцию из code.go!)
	code := GenerateRoomCode()

	// Проверяем что такой код ещё не занят
	// Если занят — генерируем новый (простая защита)
	for rooms[code] != nil {
		code = GenerateRoomCode()
	}

	// Создаём комнату
	room := &Room{
		Code:    code,
		Clients: make([]*Client, 0), // пустой список клиентов
	}

	// Сохраняем в хранилище
	rooms[code] = room

	return code
}

// GetRoom возвращает комнату по коду (или nil если не найдена)
func GetRoom(code string) *Room {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	return rooms[code]
}

// DeleteRoom удаляет комнату
func DeleteRoom(code string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	delete(rooms, code)
}

// ============================================================
// МЕТОДЫ КОМНАТЫ
// ============================================================

// AddClient добавляет клиента в комнату
func (r *Room) AddClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Clients = append(r.Clients, client)
}

// RemoveClient удаляет клиента из комнаты
func (r *Room) RemoveClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ищем клиента в списке и удаляем
	for i, c := range r.Clients {
		if c == client {
			// Удаляем элемент из среза
			// r.Clients[:i] — всё до элемента
			// r.Clients[i+1:] — всё после элемента
			// append соединяет их, пропуская удаляемый
			r.Clients = append(r.Clients[:i], r.Clients[i+1:]...)
			break
		}
	}
}

// Broadcast отправляет сообщение ВСЕМ клиентам в комнате
func (r *Room) Broadcast(message string, sender *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, client := range r.Clients {
		// Не отправляем сообщение самому отправителю
		if client != sender {
			client.Conn.Write([]byte(message))
		}
	}
}

// GetClientCount возвращает количество клиентов в комнате
func (r *Room) GetClientCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.Clients)
}
