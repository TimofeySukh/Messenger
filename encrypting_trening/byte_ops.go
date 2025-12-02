package main

import "fmt"

func main() {
	// Объявляем переменную типа byte (8 бит, значения 0-255)
	var a byte = 13 // В битах: 00001101
	var b byte = 7  // В битах: 00000111

	// XOR — символ ^
	// Сравнивает биты: разные = 1, одинаковые = 0
	xorResult := a ^ b
	fmt.Printf("XOR: %08b ^ %08b = %08b (%d)\n", a, b, xorResult, xorResult)
	// Вывод: XOR: 00001101 ^ 00000111 = 00001010 (10)
	/*
		// Сдвиг влево — символ <<
		// Двигает биты влево, справа заполняет нулями
		shiftLeft := a << 2
		fmt.Printf("Сдвиг влево: %08b << 2 = %08b (%d)\n", a, shiftLeft, shiftLeft)
		// Вывод: Сдвиг влево: 00001101 << 2 = 00110100 (52)

		// Сдвиг вправо — символ >>
		// Двигает биты вправо, слева заполняет нулями
		shiftRight := a >> 2
		fmt.Printf("Сдвиг вправо: %08b >> 2 = %08b (%d)\n", a, shiftRight, shiftRight)
		// Вывод: Сдвиг вправо: 00001101 >> 2 = 00000011 (3)

		// AND — символ &
		// Оба бита должны быть 1
		andResult := a & b
		fmt.Printf("AND: %08b & %08b = %08b (%d)\n", a, b, andResult, andResult)
		// Вывод: AND: 00001101 & 00000111 = 00000101 (5)

		// OR — символ |
		// Хотя бы один бит должен быть 1
		orResult := a | b
		fmt.Printf("OR: %08b | %08b = %08b (%d)\n", a, b, orResult, orResult)
		// Вывод: OR: 00001101 | 00000111 = 00001111 (15)

		// NOT — символ ^
		// Инвертирует все биты (в Go это XOR с 0xFF для byte)
		notResult := a ^ 0xFF
		fmt.Printf("NOT: ^%08b = %08b (%d)\n", a, notResult, notResult)
		// Вывод: NOT: ^00001101 = 11110010 (242)

	*/
}
