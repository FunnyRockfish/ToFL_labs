package leftRec

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

// Grammar представляет грамматику как структуру с картой продукций и списком нетерминалов в порядке их определения.
type Grammar struct {
	Productions      map[string][][]string
	NonTerminalOrder []string
	// Mapping для замены нетерминалов в квадратных скобках
	NonTerminalMap map[string]string
	// Счетчики для генерации новых нетерминалов
	Counter map[string]int
}

// GrammarRequest представляет структуру входного JSON-запроса.
type GrammarRequest struct {
	Productions []string `json:"productions"`
}

// GrammarResponse представляет структуру выходного JSON-ответа.
type GrammarResponse struct {
	Productions []string `json:"productions"`
}

func parseGrammar(grammarStr string) (*Grammar, error) {
	grammar := &Grammar{
		Productions:      make(map[string][][]string),
		NonTerminalOrder: []string{},
		NonTerminalMap:   make(map[string]string),
		Counter:          make(map[string]int),
	}

	scanner := bufio.NewScanner(strings.NewReader(grammarStr))
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	ruleRe := regexp.MustCompile(`^(\[[A-Za-z][A-Za-z0-9_]*\]|[A-Z][A-Za-z0-9_]*)\s*->\s*(.*)$`)

	for _, line := range lines {
		matches := ruleRe.FindStringSubmatch(line)
		if matches == nil {
			return nil, fmt.Errorf("недопустимый формат правила: %s", line)
		}
		lhs := matches[1]
		rhs := matches[2]

		originalLHS := lhs
		if strings.HasPrefix(lhs, "[") && strings.HasSuffix(lhs, "]") {
			lhs = replaceBracketNonTerminal(lhs, grammar)
		}

		if _, exists := grammar.Productions[lhs]; !exists {
			grammar.NonTerminalOrder = append(grammar.NonTerminalOrder, lhs)
		}

		alternatives := strings.Split(rhs, "|")
		for _, alt := range alternatives {
			alt = strings.TrimSpace(alt)
			var symbols []string
			if alt == "ε" {
				symbols = []string{"ε"}
			} else {
				symbols = strings.Fields(alt)
				for i, sym := range symbols {
					if strings.HasPrefix(sym, "[") && strings.HasSuffix(sym, "]") { // заменяю, чтобы было красиво)
						symbols[i] = replaceBracketNonTerminal(sym, grammar)
					}
				}
			}
			grammar.Productions[lhs] = append(grammar.Productions[lhs], symbols)
		}

		// Если lhs был в скобках, сохраняем маппинг
		if originalLHS != lhs {
			grammar.NonTerminalMap[originalLHS] = lhs
		}
	}

	return grammar, nil
}

// заменяем нетерминалы в квадратных скобках на формат A1, B2
func replaceBracketNonTerminal(nt string, grammar *Grammar) string {
	if replacement, exists := grammar.NonTerminalMap[nt]; exists {
		return replacement
	}

	base := ""
	for _, ch := range nt {
		if ch >= 'A' && ch <= 'Z' {
			base = string(ch)
			break
		}
	}
	if base == "" {
		base = "X"
	}

	grammar.Counter[base]++
	count := grammar.Counter[base]

	newNT := fmt.Sprintf("%s%d", base, count)

	for contains(grammar.NonTerminalOrder, newNT) || containsKey(grammar.Productions, newNT) {
		grammar.Counter[base]++
		newNT = fmt.Sprintf("%s%d", base, grammar.Counter[base])
	}

	grammar.NonTerminalMap[nt] = newNT

	return newNT
}

func collectGrammar(grammar *Grammar) string {
	var builder strings.Builder

	for _, nt := range grammar.NonTerminalOrder {
		productions := grammar.Productions[nt]
		var prodStrings []string
		for _, prod := range productions {
			if len(prod) == 1 && prod[0] == "ε" {
				prodStrings = append(prodStrings, "ε")
			} else {
				prodStrings = append(prodStrings, strings.Join(prod, " "))
			}
		}
		builder.WriteString(fmt.Sprintf("%s -> %s\n", nt, strings.Join(prodStrings, " | ")))
	}

	remaining := make([]string, 0)
	for nt := range grammar.Productions {
		if !contains(grammar.NonTerminalOrder, nt) {
			remaining = append(remaining, nt)
		}
	}
	sort.Strings(remaining)
	for _, nt := range remaining {
		productions := grammar.Productions[nt]
		var prodStrings []string
		for _, prod := range productions {
			if len(prod) == 1 && prod[0] == "ε" {
				prodStrings = append(prodStrings, "ε")
			} else {
				prodStrings = append(prodStrings, strings.Join(prod, " "))
			}
		}
		builder.WriteString(fmt.Sprintf("%s -> %s\n", nt, strings.Join(prodStrings, " | ")))
	}

	return strings.TrimSpace(builder.String())
}

func contains(slice []string, elem string) bool {
	for _, s := range slice {
		if s == elem {
			return true
		}
	}
	return false
}

func containsKey(m map[string][][]string, key string) bool {
	_, exists := m[key]
	return exists
}

func eliminateLeftRecursion(grammar *Grammar) {
	nonterminals := grammar.NonTerminalOrder

	for i, Ai := range nonterminals {
		// Заменяем косвенную рекурсию
		for j := 0; j < i; j++ {
			Aj := nonterminals[j]
			var newProds [][]string
			for _, production := range grammar.Productions[Ai] {
				if len(production) > 0 && production[0] == Aj {
					// Заменяем Aj его продукциями
					for _, prod := range grammar.Productions[Aj] {
						newProd := append([]string{}, prod...)
						if len(production) > 1 {
							newProd = append(newProd, production[1:]...)
						}
						newProds = append(newProds, newProd)
					}
				} else {
					newProds = append(newProds, production)
				}
			}
			grammar.Productions[Ai] = newProds
		}

		fmt.Println()

		for nt, production := range grammar.Productions {
			fmt.Println(nt, " -> ", production)
		}

		// Устраняем прямую левую рекурсию
		var recursiveProds [][]string
		var nonRecursiveProds [][]string
		for _, production := range grammar.Productions[Ai] {
			if len(production) > 0 && production[0] == Ai {
				recursiveProds = append(recursiveProds, production[1:])
			} else {
				nonRecursiveProds = append(nonRecursiveProds, production)
			}
		}

		if len(recursiveProds) > 0 {
			// Создаем новый нетерминал Ai1, Ai2 и т.д.
			base := getBaseNonTerminal(Ai)
			grammar.Counter[base]++
			AiPrime := fmt.Sprintf("%s%d", base, grammar.Counter[base])

			// Убедимся, что AiPrime уникален
			for contains(grammar.NonTerminalOrder, AiPrime) || containsKey(grammar.Productions, AiPrime) {
				grammar.Counter[base]++
				AiPrime = fmt.Sprintf("%s%d", base, grammar.Counter[base])
			}

			// Добавляем AiPrime в порядок нетерминалов
			grammar.NonTerminalOrder = append(grammar.NonTerminalOrder, AiPrime)

			// A -> β AiPrime
			var newNonRecursiveProds [][]string
			for _, prod := range nonRecursiveProds {
				newProd := append(prod, AiPrime)
				newNonRecursiveProds = append(newNonRecursiveProds, newProd)
			}
			grammar.Productions[Ai] = newNonRecursiveProds

			// AiPrime -> α AiPrime | ε
			var newRecursiveProds [][]string
			for _, prod := range recursiveProds {
				newProd := append(prod, AiPrime)
				newRecursiveProds = append(newRecursiveProds, newProd)
			}
			newRecursiveProds = append(newRecursiveProds, []string{"ε"})
			grammar.Productions[AiPrime] = newRecursiveProds
		}
	}
}

func getBaseNonTerminal(nt string) string {
	for _, ch := range nt {
		if ch >= 'A' && ch <= 'Z' {
			return string(ch)
		}
	}
	return "X" // По умолчанию
}

func leftFactor(grammar *Grammar) {
	changed := true

	for changed {
		changed = false
		newProductions := make(map[string][][]string)

		for _, A := range grammar.NonTerminalOrder {
			productions := grammar.Productions[A]
			prefix, factored := findCommonPrefix(productions)
			if len(prefix) > 0 && len(factored) > 1 {
				changed = true
				// Создаем новый нетерминал для факторизации в формате A1, A2 и т.д.
				base := getBaseNonTerminal(A)
				grammar.Counter[base]++
				AFact := fmt.Sprintf("%s%d", base, grammar.Counter[base])

				// Убедимся, что AFact уникален
				for contains(grammar.NonTerminalOrder, AFact) || containsKey(grammar.Productions, AFact) {
					grammar.Counter[base]++
					AFact = fmt.Sprintf("%s%d", base, grammar.Counter[base])
				}

				// Добавляем новый нетерминал в порядок
				grammar.NonTerminalOrder = append(grammar.NonTerminalOrder, AFact)

				// A -> prefix AFact
				newProd := append([]string{}, prefix...)
				newProd = append(newProd, AFact)
				newProductions[A] = append(newProductions[A], newProd)

				// AFact -> suffix1 | suffix2 | ...
				for _, prod := range factored {
					remainder := prod[len(prefix):]
					if len(remainder) == 0 {
						remainder = []string{"ε"}
					}
					newProductions[AFact] = append(newProductions[AFact], remainder)
				}

				// Добавляем остальные продукции, не вошедшие в факторизацию
				for _, prod := range productions {
					if !startsWithAny(factored, prod) {
						newProductions[A] = append(newProductions[A], prod)
					}
				}
			} else {
				// Если нет общего префикса, просто копируем продукцию
				newProductions[A] = append(newProductions[A], productions...)
			}
		}

		if changed {
			// Обновляем грамматику с новыми продукциями и порядком
			for nt, prods := range newProductions {
				grammar.Productions[nt] = prods
				if !contains(grammar.NonTerminalOrder, nt) {
					grammar.NonTerminalOrder = append(grammar.NonTerminalOrder, nt)
				}
			}
		}
	}
}

func findCommonPrefix(productions [][]string) ([]string, [][]string) {
	if len(productions) == 0 {
		return nil, nil
	}

	// Группируем продукции по первому символу
	prefixMap := make(map[string][][]string)
	for _, prod := range productions {
		if len(prod) == 0 {
			continue
		}
		first := prod[0]
		prefixMap[first] = append(prefixMap[first], prod)
	}

	// Ищем группу с максимальной длиной префикса
	var commonPrefix []string
	var factoredProds [][]string
	for _, prods := range prefixMap {
		if len(prods) > 1 {
			// Проверяем, есть ли общий префикс среди этих продукций
			currentPrefix := []string{prods[0][0]}
			for i := 1; ; i++ {
				if len(prods[0]) <= i {
					break
				}
				nextSymbol := prods[0][i]
				allMatch := true
				for _, p := range prods[1:] {
					if len(p) <= i || p[i] != nextSymbol {
						allMatch = false
						break
					}
				}
				if allMatch {
					currentPrefix = append(currentPrefix, nextSymbol)
				} else {
					break
				}
			}
			if len(currentPrefix) > len(commonPrefix) {
				commonPrefix = currentPrefix
				factoredProds = prods
			}
		}
	}

	if len(commonPrefix) == 0 {
		return nil, nil
	}

	return commonPrefix, factoredProds
}

// проверяем, начинается ли продукция с любой из заданных продукций.
func startsWithAny(prefixes [][]string, target []string) bool {
	for _, prefix := range prefixes {
		if len(target) >= len(prefix) && equalSlices(prefix, target[:len(prefix)]) {
			return true
		}
	}
	return false
}

func allEqual(slice []string) bool {
	if len(slice) == 0 {
		return true
	}
	first := slice[0]
	for _, s := range slice[1:] {
		if s != first {
			return false
		}
	}
	return true
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, s := range a {
		if b[i] != s {
			return false
		}
	}
	return true
}

func removeLeftRecursionAndFactor(grammar *Grammar) {
	eliminateLeftRecursion(grammar)
	leftFactor(grammar)
}

func handleTransform(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Только метод POST поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req GrammarRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	if len(req.Productions) == 0 {
		http.Error(w, "Поле 'productions' не должно быть пустым", http.StatusBadRequest)
		return
	}

	grammarStr := strings.Join(req.Productions, "\n")

	grammar, err := parseGrammar(grammarStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при парсинге грамматики: %v", err), http.StatusBadRequest)
		return
	}

	removeLeftRecursionAndFactor(grammar)

	transformedGrammarStr := collectGrammar(grammar)

	transformedProductions := strings.Split(transformedGrammarStr, "\n")

	response := GrammarResponse{
		Productions: transformedProductions,
	}

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	err = encoder.Encode(response)
	if err != nil {
		http.Error(w, "Ошибка при кодировании ответа", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/transform", handleTransform)

	fmt.Println("Сервер запущен на порту 8082. Эндпоинт: POST /transform")
	err := http.ListenAndServe(":8082", nil)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}

func TransformGrammar(productions []string) ([]string, error) {
	if len(productions) == 0 {
		return nil, fmt.Errorf("Поле 'productions' не должно быть пустым")
	}
	grammarStr := strings.Join(productions, "\n")

	grammar, err := parseGrammar(grammarStr)
	if err != nil {
		return nil, err
	}
	removeLeftRecursionAndFactor(grammar)

	// Собираем грамматику обратно в строку с сортировкой
	transformedGrammarStr := collectGrammar(grammar)

	transformedProductions := strings.Split(transformedGrammarStr, "\n")

	return transformedProductions, nil
}
