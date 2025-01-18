package grammar

import (
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

type Grammar struct {
	gram            map[string][]OneProd
	newProductions  map[string][][]string
	terminals       map[string]bool
	nonTerminals    map[string]bool
	chainProduction map[string]string
	First           map[string][]string
	follow          map[string][]string
	Last            map[string][]string
	precede         map[string][]string
	bigrammMatrix   map[string][]string
	StartSymbol     string
	Log             *zap.SugaredLogger
}

type OneProd struct {
	production []string
	isItChain  bool
}

func NewGrammar(log *zap.SugaredLogger) *Grammar {
	return &Grammar{
		gram:            make(map[string][]OneProd),
		newProductions:  make(map[string][][]string),
		terminals:       make(map[string]bool),
		nonTerminals:    make(map[string]bool),
		chainProduction: make(map[string]string),
		First:           make(map[string][]string),
		follow:          make(map[string][]string),
		Last:            make(map[string][]string),
		precede:         make(map[string][]string),
		bigrammMatrix:   make(map[string][]string),
		Log:             log,
	}
}

func (g *Grammar) GrammarParser(productions []string) {
	ruleRe := regexp.MustCompile(`^(\[[A-Za-z0-9]+\]|[A-Z]_?[0-9]*)\s*->\s*(.*)$`)
	lowercaseRe := regexp.MustCompile(`^[a-z]+$`)

	for _, line := range productions {
		match := ruleRe.FindStringSubmatch(line)
		if match == nil {
			g.Log.Errorf("Неверный формат правила: %s", line)
			continue
		}
		lhs := match[1]
		if g.StartSymbol == "" {
			g.StartSymbol = lhs
		}
		rhs := match[2]
		g.nonTerminals[lhs] = true

		alternatives := strings.Split(rhs, "|")
		for _, alt := range alternatives {
			alt = strings.TrimSpace(alt)
			altArr := strings.Fields(alt)
			g.gram[lhs] = append(g.gram[lhs], OneProd{production: altArr, isItChain: false})

			for _, symbol := range altArr {
				if symbol == "ε" {
					continue
				}
				if lowercaseRe.MatchString(symbol) {
					g.terminals[symbol] = true
				} else {
					g.nonTerminals[symbol] = true
				}
			}
		}
	}
}

func (g *Grammar) SearchChainRules() {
	g.Log.Info("Поиск цепочных правил")
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		prods, exists := g.gram[nt]
		if !exists || len(prods) == 0 {
			continue
		}

		for i := range prods {
			if len(prods[i].production) == 1 && g.isSymbolNonTerminal(prods[i].production[0]) {
				prods[i].isItChain = true
				//fmt.Println("Обнаружено цепочное правило: ", nt, " -> ", prods[i].production[0])
			}
		}
	}
}

func (g *Grammar) DeleteChainRules() {
	chainReachable := make(map[string]map[string]bool)
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		chainReachable[nt] = make(map[string]bool)
		queue := []string{nt}
		chainReachable[nt][nt] = true
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, prod := range g.gram[current] {
				if len(prod.production) == 1 && g.isSymbolNonTerminal(prod.production[0]) {
					target := prod.production[0]
					if !chainReachable[nt][target] {
						chainReachable[nt][target] = true
						queue = append(queue, target)
					}
				}
			}
		}
	}

	for _, nt := range sortedNTs {
		for target := range chainReachable[nt] {
			for _, prod := range g.gram[target] {
				if !(len(prod.production) == 1 && g.isSymbolNonTerminal(prod.production[0])) {
					if !containsProduction(g.gram[nt], prod.production) {
						g.gram[nt] = append(g.gram[nt], prod)
					}
				}
			}
		}
	}

	for _, nt := range sortedNTs {
		newProds := []OneProd{}
		for _, prod := range g.gram[nt] {
			if len(prod.production) == 1 && g.isSymbolNonTerminal(prod.production[0]) {
				// это цепная продукция, пропускаем
				continue
			}
			newProds = append(newProds, prod)
		}
		if len(newProds) != 0 {
			g.gram[nt] = newProds
		} else {
			delete(g.gram, nt)
		}
	}
}

func (g *Grammar) ConvertToCNF() error {
	g.RemoveEpsilonProductions()
	g.RemoveUnitProductions()
	g.RemoveUselessSymbols()

	if g.IsEmpty() {
		g.Log.Errorf("грамматика не производит ни одного слова")
		return fmt.Errorf("грамматика не производит ни одного слова")
	}

	g.ConvertTerminalsInProductions()
	g.ConvertLongProductions()
	g.RemoveDuplicateProductions()
	return nil
}

func (g *Grammar) RemoveEpsilonProductions() {
	nullable := make(map[string]bool)

	// находим nullable нетерминалы
	changed := true
	for changed {
		changed = false
		sortedNTs := g.sortedNonTerminals()
		for _, nt := range sortedNTs {
			if nullable[nt] {
				continue
			}
			for _, prod := range g.gram[nt] {
				if len(prod.production) == 1 && prod.production[0] == "ε" {
					nullable[nt] = true
					changed = true
					break
				}
				allNullable := true
				for _, symbol := range prod.production {
					if !g.isSymbolNonTerminal(symbol) || !nullable[symbol] {
						allNullable = false
						break
					}
				}
				if allNullable {
					nullable[nt] = true
					changed = true
					break
				}
			}
		}
	}

	// создаём новые продукции, исключая ε
	newGram := make(map[string][]OneProd)
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		for _, prod := range g.gram[nt] {
			if len(prod.production) == 1 && prod.production[0] == "ε" {
				continue
			}

			// Собираем позиции наллбл-символов
			nullableSymbols := []int{}
			for i, symbol := range prod.production {
				if g.isSymbolNonTerminal(symbol) && nullable[symbol] {
					nullableSymbols = append(nullableSymbols, i)
				}
			}
			// Все подмножества
			subsets := powerSet(nullableSymbols)
			for _, subset := range subsets {
				if len(subset) == 0 {
					continue
				}
				newProd := make([]string, 0, len(prod.production))
				for i, symbol := range prod.production {
					if !containsInt(subset, i) {
						newProd = append(newProd, symbol)
					}
				}
				if len(newProd) > 0 {
					newGram[nt] = append(newGram[nt], OneProd{production: newProd, isItChain: false})
				}
			}
			// Добавляем исходную продукцию
			newGram[nt] = append(newGram[nt], prod)
		}
	}

	if nullable[g.StartSymbol] {
		newStart := "S1"
		originalNewStart := newStart
		count := 1
		for {
			if _, exists := g.nonTerminals[newStart]; !exists {
				break
			}
			newStart = fmt.Sprintf("%s_%d", originalNewStart, count)
			count++
		}
		g.nonTerminals[newStart] = true
		newGram[newStart] = []OneProd{
			{production: []string{g.StartSymbol}},
			{production: []string{"ε"}, isItChain: false},
		}
		g.StartSymbol = newStart
	}

	g.gram = newGram
}

func (g *Grammar) RemoveUnitProductions() {
	adjacency := make(map[string][]string)
	for nt := range g.nonTerminals {
		adjacency[nt] = []string{}
	}
	for nt := range g.nonTerminals {
		for _, prod := range g.gram[nt] {
			if len(prod.production) == 1 && g.isSymbolNonTerminal(prod.production[0]) {
				adjacency[nt] = append(adjacency[nt], prod.production[0])
			}
		}
	}

	// запускаем BFS, чтобы найти все достижимые потомки
	newGram := make(map[string][]OneProd)
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		queue := []string{nt}
		visited := make(map[string]bool)
		visited[nt] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			for _, neighbor := range adjacency[current] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}

		// для всех достижимых нетерминалов добавляем нецепные продукции
		for reachableNT := range visited {
			for _, prod := range g.gram[reachableNT] {
				// Пропускаем, если это снова цепочка (A->B)
				if len(prod.production) == 1 && g.isSymbolNonTerminal(prod.production[0]) {
					continue
				}
				newGram[nt] = append(newGram[nt], prod)
			}
		}
	}

	g.gram = newGram
}

func (g *Grammar) RemoveUselessSymbols() {
	reachable := make(map[string]bool)
	reachable[g.StartSymbol] = true
	changed := true
	for changed {
		changed = false
		sortedNTs := g.sortedNonTerminals()
		for _, nt := range sortedNTs {
			if !reachable[nt] {
				continue
			}
			for _, prod := range g.gram[nt] {
				for _, symbol := range prod.production {
					if g.isSymbolNonTerminal(symbol) && !reachable[symbol] {
						reachable[symbol] = true
						changed = true
					}
				}
			}
		}
	}

	productive := make(map[string]bool)
	changed = true
	for changed {
		changed = false
		sortedNTs := g.sortedNonTerminals()
		for _, nt := range sortedNTs {
			if productive[nt] {
				continue
			}
			for _, prod := range g.gram[nt] {
				allProductive := true
				for _, symbol := range prod.production {
					if g.isSymbolNonTerminal(symbol) && !productive[symbol] {
						allProductive = false
						break
					}
				}
				if allProductive {
					productive[nt] = true
					changed = true
					break
				}
			}
		}
	}

	newGram := make(map[string][]OneProd)
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		if !reachable[nt] || !productive[nt] {
			continue
		}
		for _, prod := range g.gram[nt] {
			keep := true
			for _, symbol := range prod.production {
				if g.isSymbolNonTerminal(symbol) && (!reachable[symbol] || !productive[symbol]) {
					keep = false
					break
				}
			}
			if keep {
				newGram[nt] = append(newGram[nt], prod)
			}
		}
	}

	if !productive[g.StartSymbol] {
		g.gram = make(map[string][]OneProd)
		return
	}

	g.gram = newGram
}

func (g *Grammar) ConvertTerminalsInProductions() {
	terminalMap := make(map[string]string)
	// Для каждого терминала создаём новый нетерминал
	sortedTerminals := g.sortedTerminals()
	for _, terminal := range sortedTerminals {
		if terminal == "ε" {
			continue
		}
		if _, exists := terminalMap[terminal]; !exists {
			newNT := fmt.Sprintf("[T%s", terminal)
			originalNewNT := newNT
			count := 1
			for {
				if _, exists := g.nonTerminals[newNT]; !exists {
					break
				}
				newNT = fmt.Sprintf("%d", originalNewNT, count)
				count++
			}
			newNT += "]"
			terminalMap[terminal] = newNT
			addProduction(g, newNT, []string{terminal})
			g.nonTerminals[newNT] = true
		}
	}

	// Заменяем терминалы в продукциях длины > 1
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		for i, prod := range g.gram[nt] {
			if len(prod.production) <= 1 {
				continue
			}
			newProd := make([]string, len(prod.production))
			for j, symbol := range prod.production {
				if !g.isSymbolNonTerminal(symbol) && symbol != "ε" {
					newProd[j] = terminalMap[symbol]
				} else {
					newProd[j] = symbol
				}
			}
			g.gram[nt][i].production = newProd
		}
	}
}

func addProduction(g *Grammar, nt string, prod []string) {
	for _, existingProd := range g.gram[nt] {
		if slicesEqual(existingProd.production, prod) {
			return
		}
	}
	g.gram[nt] = append(g.gram[nt], OneProd{production: prod, isItChain: false})
}

func (g *Grammar) ConvertLongProductions() {
	intermediateMap := make(map[string]string) // для соответствий между комбинациями символов и промежуточными нетерминалами
	counter := 1
	newGram := make(map[string][]OneProd)

	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		for _, prod := range g.gram[nt] {
			if len(prod.production) <= 2 {
				newGram[nt] = append(newGram[nt], prod)
				continue
			}

			currentNT := nt
			for i := 0; i < len(prod.production)-2; i++ {
				key := currentNT + "_" + prod.production[i]
				var newSym string
				if sym, exists := intermediateMap[key]; exists {
					newSym = sym
				} else {
					newSym = fmt.Sprintf("X%d", counter)
					counter++
					intermediateMap[key] = newSym
					g.nonTerminals[newSym] = true
				}
				newGram[currentNT] = append(newGram[currentNT], OneProd{
					production: []string{prod.production[i], newSym},
					isItChain:  false,
				})
				currentNT = newSym
			}
			lastTwo := prod.production[len(prod.production)-2:]
			newGram[currentNT] = append(newGram[currentNT], OneProd{production: lastTwo, isItChain: false})
		}
	}

	g.gram = newGram
}

func (g *Grammar) RemoveDuplicateProductions() {
	sortedNTs := g.sortedNonTerminals()
	for _, nt := range sortedNTs {
		prods := g.gram[nt]
		uniqueProds := make([]OneProd, 0, len(prods))
		seen := make(map[string]bool)

		for _, prod := range prods {
			prodKey := strings.Join(prod.production, " ")
			if !seen[prodKey] {
				seen[prodKey] = true
				uniqueProds = append(uniqueProds, prod)
			}
		}
		g.gram[nt] = uniqueProds
	}
}

// //////// FIRST, FOLLOW, LAST, PRECEDE
func (g *Grammar) ComputeFirst() {
	for terminal := range g.terminals {
		g.First[terminal] = []string{terminal}
	}
	for nonTerminal := range g.nonTerminals {
		if g.First[nonTerminal] == nil {
			g.First[nonTerminal] = []string{}
		}
	}

	changed := true
	for changed {
		changed = false

		sortedNTs := g.sortedNonTerminals()
		for _, nt := range sortedNTs {
			prods := g.gram[nt]
			for _, prod := range prods {
				for i, symbol := range prod.production {
					if symbol == "ε" {
						if !contains(g.First[nt], "ε") {
							g.First[nt] = appendUnique(g.First[nt], "ε")
							changed = true
						}
						break
					} else if !g.isSymbolNonTerminal(symbol) {
						// Терминал
						if !contains(g.First[nt], symbol) {
							g.First[nt] = appendUnique(g.First[nt], symbol)
							changed = true
						}
						break
					} else {
						// Нетерминал
						for _, sym := range g.First[symbol] {
							if sym != "ε" && !contains(g.First[nt], sym) {
								g.First[nt] = appendUnique(g.First[nt], sym)
								changed = true
							}
						}
						if !contains(g.First[symbol], "ε") {
							break
						}
						if i == len(prod.production)-1 && !contains(g.First[nt], "ε") {
							g.First[nt] = appendUnique(g.First[nt], "ε")
							changed = true
						}
					}
				}
			}
		}
	}
}

func (g *Grammar) ComputeFollow() {
	g.follow[g.StartSymbol] = appendUnique(g.follow[g.StartSymbol], "$") // FOLLOW(S) = FOLLOW(S) ∪ {$}

	changed := true
	for changed {
		changed = false

		sortedNTs := g.sortedNonTerminals()
		for _, nt := range sortedNTs {
			prods := g.gram[nt]
			for _, prod := range prods {
				for i, symbol := range prod.production {
					if g.isSymbolNonTerminal(symbol) {
						before := make([]string, len(g.follow[symbol]))
						copy(before, g.follow[symbol])

						if i+1 < len(prod.production) {
							nextSymbol := prod.production[i+1]
							if !g.isSymbolNonTerminal(nextSymbol) {
								g.follow[symbol] = appendUnique(g.follow[symbol], nextSymbol) // FOLLOW(B) = FOLLOW(B) ∪ {a}
							} else { // FOLLOW(B) = FOLLOW(B) ∪ (FIRST(C) − {ε})
								for _, fs := range g.First[nextSymbol] {
									if fs != "ε" {
										g.follow[symbol] = appendUnique(g.follow[symbol], fs)
									}
								}
								if contains(g.First[nextSymbol], "ε") { // FOLLOW(B) = FOLLOW(B) ∪ FOLLOW(A)
									for _, f := range g.follow[nt] {
										g.follow[symbol] = appendUnique(g.follow[symbol], f)
									}
								}
							}
						} else {
							// Последний символ в RHS
							for _, f := range g.follow[nt] {
								g.follow[symbol] = appendUnique(g.follow[symbol], f)
							}
						}

						if !slicesEqual(before, g.follow[symbol]) {
							changed = true
						}
					}
				}
			}
		}
	}
}

// LAST(A) для A содержит все терминалы, которые могут появиться после A в любых производных строках грамматики
func (g *Grammar) ComputeLast() {
	for nt := range g.nonTerminals {
		if g.Last[nt] == nil {
			g.Last[nt] = []string{}
		}
	}

	changed := true
	for changed {
		changed = false
		sortedNTs := g.sortedNonTerminals()
		for _, nt := range sortedNTs {
			prodsArr := g.gram[nt]
			for _, prod := range prodsArr {
				symbolFound := false
				posMarker := 1

				for posMarker <= len(prod.production) && !symbolFound {
					lastSymbol := prod.production[len(prod.production)-posMarker]
					if lastSymbol == "ε" {
						posMarker++
						continue
					}

					if !g.isSymbolNonTerminal(lastSymbol) { // LAST(A)=LAST(A)∪{a}
						// Терминал
						if !contains(g.Last[nt], lastSymbol) {
							g.Last[nt] = append(g.Last[nt], lastSymbol)
							changed = true
						}
						break
					} else {
						for _, sym := range g.Last[lastSymbol] { // LAST(B) = LAST(B) ∪ LAST(A)
							if !contains(g.Last[nt], sym) {
								g.Last[nt] = append(g.Last[nt], sym)
								changed = true
							}
						}
						// Если lastSymbol не выводит ε — стоп
						if !CanGenerateEpsilon(g.First[lastSymbol]) {
							symbolFound = true
							break
						}
						posMarker++
					}
				}
			}
		}
	}
}

// Precede(A) содержит все терминалы, которые могут появиться непосредственно перед нетерминалом A в любой производной строке грамматики
func (g *Grammar) ComputePrecede() {
	for nt := range g.nonTerminals {
		if g.precede[nt] == nil {
			g.precede[nt] = []string{}
		}
	}

	changed := true
	for changed {
		changed = false
		sortedNTs := g.sortedNonTerminals()
		for _, nt := range sortedNTs {
			prodsArr := g.gram[nt]
			for _, prod := range prodsArr {
				for i, symbol := range prod.production {
					if g.isSymbolNonTerminal(symbol) {
						// B → α A β
						if i > 0 {
							prevSymbol := prod.production[i-1]
							if !g.isSymbolNonTerminal(prevSymbol) {
								//Если предыдущий символ терминал, добавляем его в PRECEDE(symbol)
								if !contains(g.precede[symbol], prevSymbol) {
									g.precede[symbol] = append(g.precede[symbol], prevSymbol)
									changed = true
								}
							} else {
								// Если предыдущий символ нетерминал, добавляем LAST(prevSymbol) в PRECEDE(symbol)
								for _, sym := range g.Last[prevSymbol] {
									if !contains(g.precede[symbol], sym) {
										g.precede[symbol] = append(g.precede[symbol], sym)
										changed = true
									}
								}
								// Если prevSymbol может выводить ε, добавляем PRECEDE(nt) в PRECEDE(symbol)
								if CanGenerateEpsilon(g.First[prevSymbol]) {
									for _, sym := range g.precede[nt] {
										if !contains(g.precede[symbol], sym) {
											g.precede[symbol] = append(g.precede[symbol], sym)
											changed = true
										}
									}
								}
							}
						} else {
							//Если символ первый в правой части продукции, добавляем ^ в PRECEDE(symbol) (нужно или нет?)
							startMarker := "^"
							if !contains(g.precede[symbol], startMarker) {
								g.precede[symbol] = append(g.precede[symbol], startMarker)
								changed = true
							}
						}
					}
				}
			}
		}
	}
}
