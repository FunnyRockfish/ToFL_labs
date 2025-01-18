package grammar

import (
	"fmt"
	"sort"
	"strings"
)

func (g *Grammar) BuildBigrammMatrix() {
	// Идём по всем парам терминалов
	sortedTerminals := g.sortedTerminals()
	for _, terminal1 := range sortedTerminals {
		for _, terminal2 := range sortedTerminals {
			if g.CheckFirstCondition(terminal1+terminal2) ||
				g.CheckSecondCondition(terminal1, terminal2) ||
				g.CheckThirdCondition(terminal1, terminal2) ||
				g.CheckFourCondition(terminal1, terminal2) {
				g.AddBigrammToMatrix(terminal1, terminal2)
			}
		}
	}
}

func (g *Grammar) CheckFirstCondition(bigramm string) bool {
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		for _, prod := range g.gram[nt] {
			prodString := strings.Join(prod.production, "")
			if strings.Contains(prodString, bigramm) {
				g.Log.Info("Первое условие выполнено для ", bigramm)
				return true
			}
		}
	}
	return false
}

func (g *Grammar) CheckSecondCondition(term1, term2 string) bool {
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		if contains(g.Last[nt], term1) && contains(g.follow[nt], term2) {
			g.Log.Info("Второе условие выполнено для ", term1, term2)
			return true
		}
	}
	return false
}

func (g *Grammar) CheckThirdCondition(term1, term2 string) bool {
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		if contains(g.precede[nt], term1) && contains(g.First[nt], term2) {
			g.Log.Info("Третье условие выполнено для ", term1, term2)
			return true
		}
	}
	return false
}

func (g *Grammar) CheckFourCondition(term1, term2 string) bool {
	sortedNTs := g.sortedNonTerminals()
	for _, A1 := range sortedNTs {
		if contains(g.Last[A1], term1) &&
			contains(g.First[A1], term2) &&
			contains(g.follow[A1], term2) {
			g.Log.Info("Четвёртое условие выполнено для ", term1, term2)
			return true
		}
	}
	return false
}

func (g *Grammar) AddBigrammToMatrix(term1, term2 string) {
	fmt.Println("Добавляем биграмму: ", term1, term2)
	g.bigrammMatrix[term1] = append(g.bigrammMatrix[term1], term2)
}

func (g *Grammar) PrintBigramm() {
	fmt.Println("\n--- Список биграмм (adjacency) ---")

	var keys []string
	for k := range g.bigrammMatrix {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, term := range keys {
		bigramms := g.bigrammMatrix[term]
		sort.Strings(bigramms)
		fmt.Printf("%s -> %v\n", term, bigramms)
	}
}

func (g *Grammar) PrintBigrammMatrix() {
	uniqueTerminals := make(map[string]bool)
	for term1, term2Set := range g.bigrammMatrix {
		uniqueTerminals[term1] = true
		for _, term2 := range term2Set {
			uniqueTerminals[term2] = true
		}
	}

	terminals := make([]string, 0, len(uniqueTerminals))
	for t := range uniqueTerminals {
		terminals = append(terminals, t)
	}
	sort.Strings(terminals)

	fmt.Println("\n--- Матрица биграмм ---")
	fmt.Printf("%-8s", " ")
	for _, term := range terminals {
		fmt.Printf("%-8s", term)
	}
	fmt.Println()

	for _, term1 := range terminals {
		fmt.Printf("%-8s", term1)
		for _, term2 := range terminals {
			if contains(g.bigrammMatrix[term1], term2) {
				fmt.Printf("%-8s", "1")
			} else {
				fmt.Printf("%-8s", "0")
			}
		}
		fmt.Println()
	}
}
