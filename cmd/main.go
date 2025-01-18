package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"lab3_1/grammar"
	"lab3_1/logger"
)

func main() {
	log := logger.CreateLogger()
	rand.Seed(time.Now().UnixNano())

	gram := grammar.NewGrammar(log)

	grammarLines := grammar.ReadInputGrammarFile("input.txt", log)
	gram.GrammarParser(grammarLines)

	log.Info("Дана грамматика:")
	gram.PrintGrammar()

	log.Info("Ищем цепные правила")
	gram.SearchChainRules()
	gram.PrintGrammar()

	log.Info("Удаляем цепные правила")
	gram.DeleteChainRules()
	gram.PrintGrammar()

	err := gram.ConvertToCNF()
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("\n--- Грамматика после преобразования в ХНФ ---")

	gram.PrintGrammar()

	gram.ComputeFirst()
	gram.ComputeFollow()
	gram.PrintGrammar()
	gram.ComputeLast()
	gram.PrintGrammar()
	gram.ComputePrecede()
	gram.PrintGrammar()
	gram.BuildBigrammMatrix()
	gram.PrintBigramm()
	gram.PrintBigrammMatrix()

	fmt.Println("\n--- Отладочная информация ---")
	fmt.Println("First(S):", gram.First[gram.StartSymbol])
	fmt.Println("Last(S):", gram.Last[gram.StartSymbol])

	numPositiveTests := 10
	numNegativeTests := 10
	maxAttempts := 100000
	wildProb := 0.5
	maxSteps := 50
	savePositive := "positive.txt"
	saveNegative := "negative.txt"
	saveLabeled := "labeled_tests.txt"

	var positiveTests []string
	var negativeTests []string
	generated := make(map[string]bool)
	attempts := 0

	for (len(positiveTests) < numPositiveTests || len(negativeTests) < numNegativeTests) && attempts < maxAttempts {
		word := gram.GenerateRandomWord(maxSteps, wildProb)
		attempts++

		if _, exists := generated[word]; exists {
			continue
		}
		generated[word] = true

		isValid := gram.ParseCYK(word)
		if isValid && len(positiveTests) < numPositiveTests {
			positiveTests = append(positiveTests, word)
			gram.Log.Infof("Тест %d: %s - Положительный", len(positiveTests), word)
		} else if !isValid && len(negativeTests) < numNegativeTests {
			negativeTests = append(negativeTests, word)
			gram.Log.Infof("Тест %d: %s - Отрицательный", len(negativeTests), word)
		}
	}

	if len(positiveTests) < numPositiveTests {
		fmt.Printf("\nНе удалось сгенерировать требуемое количество положительных тестов. Сгенерировано: %d из %d.\n",
			len(positiveTests), numPositiveTests)
	}
	if len(negativeTests) < numNegativeTests {
		fmt.Printf("\nНе удалось сгенерировать требуемое количество отрицательных тестов. Сгенерировано: %d из %d.\n",
			len(negativeTests), numNegativeTests)
	}

	grammar.SaveTestsToFile(savePositive, positiveTests)
	grammar.SaveTestsToFile(saveNegative, negativeTests)
	grammar.SaveLabeledTestsToFile(saveLabeled, positiveTests, negativeTests)

	fmt.Printf("\nСгенерировано %d положительных тестов и %d отрицательных тестов.\n", len(positiveTests), len(negativeTests))

	router := mux.NewRouter()
	router.HandleFunc("/cnf", gram.ConvertToCNFHandler).Methods("POST")
	router.HandleFunc("/test", gram.GenerateTestsHandler).Methods("POST")
	fmt.Println("Сервер запущен на порту 8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}
}
