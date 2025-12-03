package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func main() {
	// ==========================================
	// ШАГ 1: Создаём слушатель
	// ==========================================

	listener, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer listener.Close()

	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║     SERVER RUNNING (port 8080)     ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("Waiting for connections...")
	fmt.Println("")

	// ==========================================
	// ШАГ 2: Бесконечный цикл — принимаем клиентов
	// ==========================================

	// for {} — бесконечный цикл
	// Сервер постоянно ждёт новых клиентов
	for {
		// Accept ждёт нового подключения
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue // пропускаем ошибку, ждём следующего
		}

		// go handleClient(conn) — запускаем обработку в ГОРУТИНЕ
		//
		// Горутина = легковесный поток
		// Это позволяет обрабатывать много клиентов ОДНОВРЕМЕННО
		//
		// Без "go": сервер обслужит одного клиента, потом следующего
		// С "go":   сервер обслуживает всех одновременно
		go handleClient(conn)
	}
}

// handleClient обрабатывает одного клиента
// Эта функция запускается в отдельной горутине для каждого клиента
func handleClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// ==========================================
	// ШАГ 1: Получаем username
	// ==========================================

	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading username:", err)
		return
	}
	username = strings.TrimSpace(username)

	fmt.Printf("→ New connection: %s\n", username)

	// ==========================================
	// ШАГ 2: Получаем команду (create или connect)
	// ==========================================

	command, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading command:", err)
		return
	}
	command = strings.TrimSpace(command)

	var room *Room
	var client *Client

	// ==========================================
	// ШАГ 3: Обрабатываем команду
	// ==========================================

	if command == "create" {
		// CREATE — создаём новую комнату

		// Вызываем функцию из room.go
		code := CreateRoom()

		// Получаем созданную комнату
		room = GetRoom(code)

		// Создаём объект клиента
		client = &Client{
			Conn:     conn,
			Username: username,
		}

		// Добавляем клиента в комнату
		room.AddClient(client)

		// Отправляем код клиенту
		// Формат: "CODE:12345\n"
		conn.Write([]byte("CODE:" + code + "\n"))

		fmt.Printf("✓ Room created: %s by %s\n", code, username)

	} else if command == "connect" {
		// CONNECT — подключаемся к существующей комнате

		// Читаем код от клиента
		code, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading code:", err)
			return
		}
		code = strings.TrimSpace(code)

		// Ищем комнату
		room = GetRoom(code)

		if room == nil {
			// Комната не найдена
			conn.Write([]byte("ERROR:Room not found\n"))
			fmt.Printf("✗ Room not found: %s (requested by %s)\n", code, username)
			return
		}

		// Создаём объект клиента
		client = &Client{
			Conn:     conn,
			Username: username,
		}

		// Добавляем в комнату
		room.AddClient(client)

		// Отправляем подтверждение
		conn.Write([]byte("OK:Connected to room " + code + "\n"))

		// Уведомляем остальных в комнате
		room.Broadcast(fmt.Sprintf(">>> %s joined the room\n", username), client)

		fmt.Printf("✓ %s joined room: %s\n", username, code)

	} else {
		// Неизвестная команда
		conn.Write([]byte("ERROR:Unknown command. Use 'create' or 'connect'\n"))
		return
	}

	// ==========================================
	// ШАГ 4: Режим чата — читаем и рассылаем сообщения
	// ==========================================

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			// Клиент отключился
			fmt.Printf("← %s disconnected from room %s\n", username, room.Code)

			// Удаляем из комнаты
			room.RemoveClient(client)

			// Уведомляем остальных
			room.Broadcast(fmt.Sprintf("<<< %s left the room\n", username), client)

			// Если комната пустая — удаляем её
			if room.GetClientCount() == 0 {
				DeleteRoom(room.Code)
				fmt.Printf("✗ Room deleted: %s (empty)\n", room.Code)
			}

			return
		}

		// Формируем сообщение с именем отправителя
		formattedMessage := fmt.Sprintf("[%s] %s", username, message)

		// Рассылаем всем в комнате (кроме отправителя)
		room.Broadcast(formattedMessage, client)

		// Логируем на сервере
		fmt.Printf("[%s] %s: %s", room.Code, username, message)
	}
}
