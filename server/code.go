package main

// ============================================================
// ЭТОТ КОД — КОПИЯ random.go (твоя логика!)
// Изменения только в GenerateRoomCode() в конце файла
// ============================================================

import (
	"fmt"
	"math/rand"
	"time"
)

// init вызывается автоматически при старте программы
func init() {
	rand.Seed(time.Now().UnixNano())
}

// Глобальные переменные для часовых поясов
var (
	helsinkiLoc = mustLoadLocation("Europe/Helsinki")
	naplesLoc   = mustLoadLocation("Europe/Rome")
	londonLoc   = mustLoadLocation("Europe/London")
	sfLoc       = mustLoadLocation("America/Los_Angeles")
)

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// тут начинаем собирать data
func getHelsinkiHour() byte {
	loc, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		loc = time.UTC
	}
	return byte(time.Now().In(loc).Hour())
}

func getNaplesHour() byte {
	loc, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		loc = time.UTC
	}
	return byte(time.Now().In(loc).Hour())
}

func getLondonHour() byte {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		loc = time.UTC
	}
	return byte(time.Now().In(loc).Hour())
}

func getSanFranciscoUnixMinutes() byte {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.UTC
	}
	unixMinutes := time.Now().In(loc).Unix() / 60
	return byte(unixMinutes) // Берём только последний байт
}

func getNanoseconds() byte {
	return byte(time.Now().Nanosecond())
}

func Data() byte {
	neapol := getNaplesHour()
	london := getLondonHour()
	helsinki := getHelsinkiHour()
	nano := getNanoseconds()
	san_francisco := getSanFranciscoUnixMinutes()
	key := neapol ^ london ^ helsinki ^ san_francisco ^ nano
	return key
}

func Key() byte {
	number := rand.Intn(100) + 10
	var result_key int
	for i := 0; i < number; i++ {
		num1 := rand.Intn(10000)
		num2 := rand.Intn(10000)
		result_key += num1 ^ num2
	}
	return byte(result_key)
}

// ============================================================
// ГЕНЕРАЦИЯ 8-ЗНАЧНОГО КОДА КОМНАТЫ
// Использует ТВОИ функции Data() и Key()
// ============================================================

// GenerateRoomCode создаёт 8-значный код комнаты
//
// Как работает:
//  1. Вызываем Data() ^ Key() — получаем 1 байт (0-255)
//  2. Делаем это 3 раза — получаем 3 байта
//  3. Объединяем: byte1 * 65536 + byte2 * 256 + byte3
//     (это как записать число в 256-ричной системе)
//  4. Берём остаток от 100000000 (чтобы было 8 цифр)
//  5. Форматируем с ведущими нулями
func GenerateRoomCode() string {
	// Вызываем твои функции 3 раза
	byte1 := Data() ^ Key()
	byte2 := Data() ^ Key()
	byte3 := Data() ^ Key()

	// Объединяем в одно число
	// byte1 = 0-255, byte2 = 0-255, byte3 = 0-255
	// combined = 0 до 16,777,215 (256 * 256 * 256 - 1)
	combined := int(byte1)*65536 + int(byte2)*256 + int(byte3)

	// Берём остаток от 100,000,000 чтобы получить 8-значное число
	code := combined % 100000000

	// Форматируем: 42 → "00000042"
	// %08d = 8 цифр, с ведущими нулями
	return fmt.Sprintf("%08d", code)
}

// IsValidCode проверяет формат кода
func IsValidCode(code string) bool {
	// Проверяем длину — должно быть ровно 8 символов
	if len(code) != 8 {
		return false
	}

	// Проверяем что все символы — цифры
	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
