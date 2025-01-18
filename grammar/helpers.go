package grammar

import (
	"fmt"
	"sort"
	"strings"
)

func (g *Grammar) isSymbolNonTerminal(symbol string) bool {
	_, exists := g.nonTerminals[symbol]
	return exists
}

func (g *Grammar) sortedNonTerminals() []string {
	var nts []string
	for nt := range g.nonTerminals {
		nts = append(nts, nt)
	}
	sort.Strings(nts)
	return nts
}

func (g *Grammar) sortedTerminals() []string {
	var ts []string
	for t := range g.terminals {
		ts = append(ts, t)
	}
	sort.Strings(ts)
	return ts
}

func containsProduction(productions []OneProd, newProd []string) bool {
	for _, prod := range productions {
		if slicesEqual(prod.production, newProd) {
			return true
		}
	}
	return false
}

func appendUnique(slice []string, elem string) []string {
	for _, v := range slice {
		if v == elem {
			return slice
		}
	}
	return append(slice, elem)
}

func contains(slice []string, elem string) bool {
	for _, v := range slice {
		if v == elem {
			return true
		}
	}
	return false
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	bMap := make(map[string]bool)
	for _, v := range a {
		aMap[v] = true
	}
	for _, v := range b {
		bMap[v] = true
	}
	for k := range aMap {
		if !bMap[k] {
			return false
		}
	}
	return true
}

func (g *Grammar) IsEmpty() bool {
	for _, prods := range g.gram {
		if len(prods) > 0 {
			return false
		}
	}
	return true
}

func powerSet(set []int) [][]int {
	n := len(set)
	result := [][]int{}
	for i := 0; i < (1 << n); i++ { // гламур для числа 2^n - 1 (так как для среза длины n эл-тов кол-во подмножеств 2^n
		subset := []int{}
		for j := 0; j < n; j++ {
			if i&(1<<j) != 0 { // побитовое И между i и 2^j (если 0 - то эл-т set[j] включён в текущее подмножество)
				subset = append(subset, set[j])
			}
		}
		result = append(result, subset)
	}
	return result
}

func containsInt(slice []int, elem int) bool {
	for _, v := range slice {
		if v == elem {
			return true
		}
	}
	return false
}

func CanGenerateEpsilon(firstSet []string) bool {
	for _, sym := range firstSet {
		if sym == "ε" {
			return true
		}
	}
	return false
}

func (g *Grammar) PrintGrammar() {
	fmt.Println("\n--- Грамматика ---")

	sortedNTs := g.sortedNonTerminals()

	if _, ok := g.gram[g.StartSymbol]; ok {
		fmt.Printf("%s -> ", g.StartSymbol)
		prods := g.gram[g.StartSymbol]
		for i, oneProd := range prods {
			if i > 0 {
				fmt.Print(" | ")
			}
			fmt.Print(strings.Join(oneProd.production, " "))
		}
		fmt.Println()
	}

	for _, nt := range sortedNTs {
		if nt == g.StartSymbol {
			continue
		}
		prods, exists := g.gram[nt]
		if !exists || len(prods) == 0 {
			continue
		}

		fmt.Printf("%s -> ", nt)
		for i, oneProd := range prods {
			if i > 0 {
				fmt.Print(" | ")
			}
			fmt.Print(strings.Join(oneProd.production, " "))
		}
		fmt.Println()
	}

	fmt.Println("Нетерминалы:")
	fmt.Print(strings.Join(sortedNTs, " "))
	fmt.Println()

	fmt.Println("\nЦепочные правила (isItChain=true):")
	for _, nt := range sortedNTs {
		prods, exists := g.gram[nt]
		if !exists || len(prods) == 0 {
			continue
		}
		for _, oneProd := range prods {
			if oneProd.isItChain {
				fmt.Printf("%s -> %s\n", nt, strings.Join(oneProd.production, " "))
			}
		}
	}

	fmt.Println("\nFirst:")
	for _, nt := range sortedNTs {
		fmt.Println(nt, ":", g.First[nt])
	}
	fmt.Println("Follow:")
	for _, nt := range sortedNTs {
		fmt.Println(nt, ":", g.follow[nt])
	}

	fmt.Println("Last:")
	for _, nt := range sortedNTs {
		fmt.Println(nt, ":", g.Last[nt])
	}

	fmt.Println("Precede:")
	for _, nt := range sortedNTs {
		fmt.Println(nt, ":", g.precede[nt])
	}
}

func (g *Grammar) collectProductionsInOrder() ([]string, error) {
	sortedNTs := g.sortedNonTerminals()
	indexOfStart := -1
	for i, nt := range sortedNTs {
		if nt == g.StartSymbol {
			indexOfStart = i
			break
		}
	}
	if indexOfStart >= 0 {
		// Перенесём startSymbol на позицию 0
		sortedNTs = append([]string{g.StartSymbol}, append(sortedNTs[:indexOfStart], sortedNTs[indexOfStart+1:]...)...)
	}

	var responseProductions []string

	for _, nt := range sortedNTs {
		prods, exists := g.gram[nt]
		if !exists {
			continue
		}
		if len(prods) == 0 {
			continue
		}

		var prodStrings []string
		for _, prod := range prods {
			prodStr := strings.Join(prod.production, " ")
			prodStrings = append(prodStrings, prodStr)
		}
		combinedProds := strings.Join(prodStrings, " | ")
		combinedRule := fmt.Sprintf("%s -> %s", nt, combinedProds)
		responseProductions = append(responseProductions, combinedRule)
	}

	return responseProductions, nil
}
