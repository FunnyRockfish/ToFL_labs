package grammar

import (
	"bufio"
	"fmt"
	"os"
)

func SaveTestsToFile(filename string, tests []string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Ошибка при создании файла %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, test := range tests {
		_, err := writer.WriteString(test + "\n")
		if err != nil {
			fmt.Printf("Ошибка при записи в файл %s: %v\n", filename, err)
			return
		}
	}
	writer.Flush()
	fmt.Printf("Тесты сохранены в файл %s\n", filename)
}

func SaveLabeledTestsToFile(filename string, positiveTests, negativeTests []string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Ошибка при создании файла %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, test := range positiveTests {
		_, err := writer.WriteString(test + " 1\n")
		if err != nil {
			fmt.Printf("Ошибка при записи в файл %s: %v\n", filename, err)
			return
		}
	}
	for _, test := range negativeTests {
		_, err := writer.WriteString(test + " 0\n")
		if err != nil {
			fmt.Printf("Ошибка при записи в файл %s: %v\n", filename, err)
			return
		}
	}
	writer.Flush()
	fmt.Printf("Тесты с метками сохранены в файл %s\n", filename)
}
