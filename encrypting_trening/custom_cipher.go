package main

import (
	"encoding/hex"
	"fmt"
)

// ============================================================
// КАСТОМНОЕ ШИФРОВАНИЕ (Сложность: 6/10)
// ЭТО НЕ БЕЗОПАСНО для реального использования!
// Только для обучения
// ============================================================

// S-Box — таблица замен (256 элементов)
// Каждый байт заменяется на другой по этой таблице
// Это добавляет "нелинейность" — ломает математические зависимости
var sBox [256]byte
var sBoxInverse [256]byte // обратная таблица для расшифровки

// Инициализация S-Box
func initSBox() {
	// Простая но рабочая генерация
	for i := 0; i < 256; i++ {
		// Формула: (i * 167 + 53) mod 256
		// 167 — взаимно простое с 256, поэтому все значения уникальны
		val := byte((i*167 + 53) % 256)
		sBox[i] = val
		sBoxInverse[val] = byte(i)
	}
}

// Циклический сдвиг влево
// Биты которые "выпадают" слева появляются справа
func rotateLeft(b byte, n uint) byte {
	n = n % 8 // защита от сдвига больше чем на 7
	return (b << n) | (b >> (8 - n))
}

// Циклический сдвиг вправо
func rotateRight(b byte, n uint) byte {
	n = n % 8
	return (b >> n) | (b << (8 - n))
}

// Один раунд шифрования
// Раунд = набор операций который повторяется несколько раз
func encryptRound(data []byte, roundKey []byte, roundNum int) []byte {
	result := make([]byte, len(data))

	for i := 0; i < len(data); i++ {
		b := data[i]

		// Шаг 1: XOR с ключом раунда
		// XOR — обратим сам собой: (a ^ k) ^ k = a
		b = b ^ roundKey[i%len(roundKey)]

		// Шаг 2: Подстановка через S-Box
		// Заменяем байт по таблице
		b = sBox[b]

		// Шаг 3: Циклический сдвиг
		// Величина сдвига зависит от позиции и номера раунда
		shift := uint((i+roundNum)%7) + 1 // от 1 до 7
		b = rotateLeft(b, shift)

		// Шаг 4: Ещё один XOR с другой частью ключа
		b = b ^ roundKey[(i+roundNum+5)%len(roundKey)]

		result[i] = b
	}

	return result
}

// Один раунд расшифровки (операции в обратном порядке!)
func decryptRound(data []byte, roundKey []byte, roundNum int) []byte {
	result := make([]byte, len(data))

	for i := 0; i < len(data); i++ {
		b := data[i]

		// Шаг 4 → 1 (обратный порядок!)

		// Обратный шаг 4: XOR
		b = b ^ roundKey[(i+roundNum+5)%len(roundKey)]

		// Обратный шаг 3: Сдвиг в другую сторону
		shift := uint((i+roundNum)%7) + 1
		b = rotateRight(b, shift)

		// Обратный шаг 2: Обратная подстановка
		b = sBoxInverse[b]

		// Обратный шаг 1: XOR
		b = b ^ roundKey[i%len(roundKey)]

		result[i] = b
	}

	return result
}

// Количество раундов
// Больше раундов = сложнее взломать, но медленнее
const ROUNDS = 8

// Полное шифрование
func encrypt(plaintext []byte, key []byte) []byte {
	initSBox()

	data := make([]byte, len(plaintext))
	copy(data, plaintext)

	// Применяем раунды последовательно: 0, 1, 2, ... 7
	for r := 0; r < ROUNDS; r++ {
		data = encryptRound(data, key, r)
	}

	return data
}

// Полная расшифровка
func decrypt(ciphertext []byte, key []byte) []byte {
	initSBox()

	data := make([]byte, len(ciphertext))
	copy(data, ciphertext)

	// Раунды в ОБРАТНОМ порядке: 7, 6, 5, ... 0
	for r := ROUNDS - 1; r >= 0; r-- {
		data = decryptRound(data, key, r)
	}

	return data
}

func main() {
	fmt.Println("=== КАСТОМНЫЙ ШИФР (6/10 сложность) ===")
	fmt.Println("⚠️  Только для обучения, НЕ безопасно!")
	fmt.Println()

	// Ключ (должен быть секретным)
	key := []byte("MySecretKey12345") // 16 байт = 128 бит
	fmt.Printf("Ключ: %s\n", key)
	fmt.Printf("Ключ (hex): %s\n", hex.EncodeToString(key))
	fmt.Println()

	// Сообщение
	message := []byte("hello")
	fmt.Printf("Оригинал: %s\n", message)
	fmt.Printf("Оригинал (hex): %s\n", hex.EncodeToString(message))
	fmt.Println()

	// Шифруем
	encrypted := encrypt(message, key)
	fmt.Printf("Зашифровано (hex): %s\n", hex.EncodeToString(encrypted))
	fmt.Println()

	// Расшифровываем
	decrypted := decrypt(encrypted, key)
	fmt.Printf("Расшифровано: %s\n", decrypted)
	fmt.Println()

	// Демонстрация эффекта лавины
	fmt.Println("=== ЭФФЕКТ ЛАВИНЫ ===")
	message2 := []byte("hfllo") // изменили одну букву
	encrypted2 := encrypt(message2, key)
	fmt.Printf("'hello' → %s\n", hex.EncodeToString(encrypted))
	fmt.Printf("'hfllo' → %s\n", hex.EncodeToString(encrypted2))
	fmt.Println("↑ Одна буква изменилась — результат совсем другой!")
}
