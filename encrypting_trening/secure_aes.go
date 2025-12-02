package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// ============================================================
// БЕЗОПАСНОЕ ШИФРОВАНИЕ: AES-256-GCM
// Это индустриальный стандарт, используется везде:
// WhatsApp, Signal, банки, военные
// ============================================================

// Шифрование
// Принимает: plaintext (исходный текст), key (ключ 32 байта = 256 бит)
// Возвращает: зашифрованные данные, ошибку
func encryptAES(plaintext []byte, key []byte) ([]byte, error) {
	// aes.NewCipher создаёт "блочный шифр" из ключа
	// Блочный = шифрует данные блоками по 16 байт
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// GCM = Galois/Counter Mode
	// Это "режим работы" шифра — как именно шифровать много блоков
	// GCM также проверяет целостность (никто не изменил данные)
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Nonce = Number used ONCE (одноразовое число)
	// Делает каждое шифрование уникальным даже с тем же ключом
	// КРИТИЧНО: никогда не использовать один nonce дважды с тем же ключом!
	nonce := make([]byte, aesGCM.NonceSize()) // обычно 12 байт
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal шифрует и добавляет тег аутентификации
	// nonce добавляем в начало результата чтобы использовать при расшифровке
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Расшифровка
func decryptAES(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Извлекаем nonce из начала зашифрованных данных
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Open расшифровывает и проверяет целостность
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// Генерация безопасного случайного ключа
func generateKey() []byte {
	key := make([]byte, 32) // 32 байта = 256 бит
	rand.Read(key)          // crypto/rand — криптографически безопасный
	return key
}

func main() {
	fmt.Println("=== AES-256-GCM (БЕЗОПАСНОЕ) ===")
	fmt.Println()

	// Генерируем случайный ключ
	key := generateKey()
	fmt.Printf("Ключ (hex): %s\n", hex.EncodeToString(key))
	fmt.Printf("Длина ключа: %d байт = %d бит\n", len(key), len(key)*8)
	fmt.Println()

	// Наше сообщение
	message := []byte("hello")
	fmt.Printf("Оригинал: %s\n", message)
	fmt.Printf("Оригинал (hex): %s\n", hex.EncodeToString(message))
	fmt.Println()

	// Шифруем
	encrypted, err := encryptAES(message, key)
	if err != nil {
		fmt.Println("Ошибка шифрования:", err)
		return
	}
	fmt.Printf("Зашифровано (hex): %s\n", hex.EncodeToString(encrypted))
	fmt.Printf("Длина: %d байт\n", len(encrypted))
	fmt.Println()

	// Расшифровываем
	decrypted, err := decryptAES(encrypted, key)
	if err != nil {
		fmt.Println("Ошибка расшифровки:", err)
		return
	}
	fmt.Printf("Расшифровано: %s\n", decrypted)
}
