package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func errCheck(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1) // 1 = ошибка, 0 = успех
	}
}
func checkServer(address string) bool {
	// Пробуем подключиться с таймаутом 2 секунды
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)

	if err != nil {
		// Не удалось подключиться
		return false
	}

	// Удалось! Закрываем соединение
	conn.Close()
	return true
}
func main() {
	// ==========================================
	// ШАГ 1: Получаем IP из флага или переменной окружения
	// ==========================================

	// flag.String создаёт флаг:
	//   "ip"  — имя флага (будет -ip)
	//   ""    — значение по умолчанию (пустая строка)
	//   "..." — описание для --help
	flagIP := flag.String("ip", "", "Server IP address (e.g. 192.168.1.100:8080)")
	flag.Parse() // Читает аргументы командной строки

	// Получаем IP: сначала из флага, если нет — из переменной окружения
	var serverIP string
	if *flagIP != "" {
		// IP передан через флаг -ip
		serverIP = *flagIP
	} else {
		// Пробуем взять из переменной окружения SERVER_IP
		serverIP = os.Getenv("SERVER_IP")
	}

	// ==========================================
	// ШАГ 2: Проверяем что IP передан
	// ==========================================

	if serverIP == "" {
		fmt.Println("Error: Server IP is required")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  Method 1: Flag")
		fmt.Println("    go run client.go -ip=192.168.1.100:8080")
		fmt.Println("")
		fmt.Println("  Method 2: Environment variable")
		fmt.Println("    export SERVER_IP=192.168.1.100:8080")
		fmt.Println("    go run client.go")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  go run client.go -ip=192.168.1.100:8080")
		fmt.Println("  export SERVER_IP=100.64.0.5:8080 && go run client.go")
		os.Exit(1)
	}

	// Добавляем порт если не указан
	if !strings.Contains(serverIP, ":") {
		serverIP = serverIP + ":8080"
	}

	// ==========================================
	// ШАГ 3: Проверяем доступность сервера
	// ==========================================

	fmt.Println("Checking server", serverIP, "...")
	if !checkServer(serverIP) {
		fmt.Println("Error: Cannot reach server at", serverIP)
		fmt.Println("")
		fmt.Println("Check:")
		fmt.Println("  1. Is server running?")
		fmt.Println("  2. Is IP address correct?")
		fmt.Println("  3. Are you on the same network?")
		os.Exit(1)
	}
	fmt.Println("Server is reachable!")
	fmt.Println("")

	// ==========================================
	// ШАГ 4: Красивое приветствие
	// ==========================================

	inputReader := bufio.NewReader(os.Stdin)

	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║          SIMPLE MESSENGER          ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println("")

	// ==========================================
	// ШАГ 5: Ввод username
	// ==========================================

	var username string

	for {
		fmt.Print("Enter your username: ")
		var err error
		username, err = inputReader.ReadString('\n')
		errCheck(err)
		username = strings.TrimSpace(username)

		if username == "" {
			fmt.Println("⚠ Username cannot be empty! Try again.")

		} else if len(username) > 10 {
			fmt.Println("Meh tf is that username should be maximum 10 chars")

		} else {
			fmt.Println("Accepted")
			break
		}
	}

	fmt.Println("")

	// ==========================================
	// ШАГ 6: Меню — create или connect
	// ==========================================

	var command string

	for {
		fmt.Println("What do you want to do?")
		fmt.Println("  [1] create  - Create a new room")
		fmt.Println("  [2] connect - Join existing room")
		fmt.Println("")
		fmt.Print("Your choice: ")

		choice, err := inputReader.ReadString('\n')
		errCheck(err)
		choice = strings.TrimSpace(choice)

		if choice == "1" || choice == "create" {
			command = "create"
			break
		} else if choice == "2" || choice == "connect" {
			command = "connect"
			break
		} else {
			fmt.Println("⚠ Invalid choice! Please enter 1 or 2.")
		}
	}

	fmt.Println("")

	// ==========================================
	// ШАГ 7: Подключаемся к серверу
	// ==========================================

	fmt.Println("Connecting to", serverIP, "...")

	conn, err := net.Dial("tcp", serverIP)
	if err != nil {
		fmt.Println("Error: Connection failed:", err)
		os.Exit(1)
	}
	defer conn.Close()

	serverReader := bufio.NewReader(conn)

	// ==========================================
	// ШАГ 8: Отправляем username и команду
	// ==========================================

	conn.Write([]byte(username + "\n"))
	conn.Write([]byte(command + "\n"))

	// ==========================================
	// ШАГ 9: Обрабатываем ответ сервера
	// ==========================================

	if command == "create" {
		response, err := serverReader.ReadString('\n')
		errCheck(err)
		response = strings.TrimSpace(response)

		if strings.HasPrefix(response, "CODE:") {
			roomCode := strings.TrimPrefix(response, "CODE:")
			fmt.Println("╔════════════════════════════════════╗")
			fmt.Println("║         ROOM CREATED!              ║")
			fmt.Println("╠════════════════════════════════════╣")
			fmt.Printf("║   Code: %-26s║\n", roomCode)
			fmt.Println("║   Share this code with friends!    ║")
			fmt.Println("╚════════════════════════════════════╝")
		} else {
			fmt.Println("Unexpected response:", response)
			return
		}

	} else if command == "connect" {
		// Ввод кода комнаты (с повтором при ошибке)
		var roomCode string

		for {
			fmt.Print("Enter room code (8 digits): ")
			var err error
			roomCode, err = inputReader.ReadString('\n')
			errCheck(err)
			roomCode = strings.TrimSpace(roomCode)

			if len(roomCode) == 8 {
				break
			}

			fmt.Println("⚠ Room code must be exactly 8 digits! Try again.")
		}

		conn.Write([]byte(roomCode + "\n"))

		response, err := serverReader.ReadString('\n')
		errCheck(err)
		response = strings.TrimSpace(response)

		if strings.HasPrefix(response, "ERROR:") {
			errorMsg := strings.TrimPrefix(response, "ERROR:")
			fmt.Println("✗ Error:", errorMsg)
			return
		}

		fmt.Println("╔════════════════════════════════════╗")
		fmt.Println("║       CONNECTED TO ROOM!           ║")
		fmt.Println("╚════════════════════════════════════╝")
	}

	fmt.Println("")
	fmt.Println("Type messages and press Enter.")
	fmt.Println("To exit: Ctrl+C")
	fmt.Println("")

	// ==========================================
	// ШАГ 10: Горутина для получения сообщений
	// ==========================================

	go func() {
		for {
			message, err := serverReader.ReadString('\n')
			errCheck(err)
			fmt.Print(message)
		}
	}()

	// ==========================================
	// ШАГ 11: Основной цикл — отправка сообщений
	// ==========================================

	for {
		message, err := inputReader.ReadString('\n')
		errCheck(err)

		_, err = conn.Write([]byte(message))
		errCheck(err)
	}
}
