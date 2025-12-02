package main

import (
	"fmt"
	"strconv"
)

func main() {
	// ПРАВИЛЬНЫЙ двоичный код для "hello" (каждая буква = 8 бит)
	// Важно: каждая буква должна быть РОВНО 8 символов!
	binaryString := "0110100001100101011011000110110001101111"
	//               ^^^^^^^^ ^^^^^^^^ ^^^^^^^^ ^^^^^^^^ ^^^^^^^^
	//                  h        e        l        l        o
	//                 104      101      108      108      111

	fmt.Println("Двоичный код:", binaryString)
	fmt.Println("Длина:", len(binaryString), "символов")
	fmt.Println("---")

	// Проверяем что длина кратна 8
	if len(binaryString)%8 != 0 {
		fmt.Println("ОШИБКА: длина должна быть кратна 8!")
		fmt.Println("У тебя:", len(binaryString), "символов")
		fmt.Println("Не хватает:", 8-len(binaryString)%8, "нулей в начале")
		return
	}

	result := ""

	// Идём по строке шагами по 8 символов
	for i := 0; i < len(binaryString); i += 8 {
		// Вырезаем 8 символов начиная с позиции i
		// binaryString[i:i+8] — это "срез" строки
		bits := binaryString[i : i+8]

		// Превращаем двоичную строку в число
		number, err := strconv.ParseInt(bits, 2, 8)
		if err != nil {
			fmt.Println("Ошибка:", err)
			continue
		}

		letter := string(byte(number))
		fmt.Printf("%s → %3d → '%s'\n", bits, number, letter)

		result += letter
	}

	fmt.Println("---")
	fmt.Println("Результат:", result)
}
