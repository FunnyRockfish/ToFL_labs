package grammar

import (
	"math/rand"
	"strings"
)

func (g *Grammar) GenerateRandomWord(maxSteps int, wildProb float64) string {
	if CanGenerateEpsilon(g.First[g.StartSymbol]) {
		if rand.Float64() < 0.1 {
			return ""
		}
	}

	startTerminals := g.First[g.StartSymbol]
	// Исключаем ε
	filteredStartTerminals := []string{}
	for _, t := range startTerminals {
		if t != "ε" {
			filteredStartTerminals = append(filteredStartTerminals, t)
		}
	}

	var allTerminals []string
	for t := range g.terminals {
		if t != "ε" {
			allTerminals = append(allTerminals, t)
		}
	}

	if len(filteredStartTerminals) == 0 {
		// Если FIRST(S) пуст или содержит только ε
		return ""
	}

	current := pickRandom(filteredStartTerminals)
	word := []string{current}

	for i := 1; i < maxSteps; i++ {
		// 10% шанс остановиться, если текущий терминал входит в last(S)
		if contains(g.Last[g.StartSymbol], current) && rand.Float64() < 0.1 {
			break
		}

		// «Шальный» ход
		if rand.Float64() < wildProb {
			current = pickRandom(allTerminals)
			word = append(word, current)
			continue
		}

		nextCandidates := g.bigrammMatrix[current]
		if len(nextCandidates) == 0 {
			break
		}
		current = pickRandom(nextCandidates)
		word = append(word, current)
	}

	return strings.Join(word, "")
}

func pickRandom(elements []string) string {
	if len(elements) == 0 {
		return ""
	}
	return elements[rand.Intn(len(elements))]
}
