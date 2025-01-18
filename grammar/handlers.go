package grammar

import (
	"encoding/json"
	"fmt"
	"net/http"

	"lab3_1/domain"
	"lab3_1/leftRec"
)

func (g *Grammar) ConvertToCNFHandler(w http.ResponseWriter, r *http.Request) {
	var req domain.GrammarRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(req.Productions) == 0 {
		http.Error(w, "Грамматика не может быть пустой", http.StatusBadRequest)
		return
	}

	g.gram = make(map[string][]OneProd)
	g.newProductions = make(map[string][][]string)
	g.terminals = make(map[string]bool)
	g.nonTerminals = make(map[string]bool)
	g.chainProduction = make(map[string]string)
	g.First = make(map[string][]string)
	g.follow = make(map[string][]string)
	g.Last = make(map[string][]string)
	g.precede = make(map[string][]string)
	g.bigrammMatrix = make(map[string][]string)
	g.StartSymbol = ""

	cleanProd, err := leftRec.TransformGrammar(req.Productions)
	if err != nil {
		g.Log.Error(err)
		http.Error(w, "проблемы при удалении левой рекурсии и факторизации", http.StatusBadRequest)
		return
	}

	g.GrammarParser(cleanProd)
	g.PrintGrammar()
	g.SearchChainRules()
	g.DeleteChainRules()
	g.RemoveUselessSymbols()
	if g.IsEmpty() {
		g.Log.Errorf("грамматика не производит ни одного слова")
		http.Error(w, "Грамматика не может быть пустой", http.StatusBadRequest)
		return
	}
	g.PrintGrammar()

	// Собираем детерминированный порядок продукций
	responseProductions, err := g.collectProductionsInOrder()
	if err != nil {
		g.Log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := domain.GrammarResponse{
		Productions: responseProductions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (g *Grammar) GenerateTestsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)

	var req domain.TestReq

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(req.Productions) == 0 {
		http.Error(w, "Грамматика не может быть пустой", http.StatusBadRequest)
		return
	}

	g.gram = make(map[string][]OneProd)
	g.newProductions = make(map[string][][]string)
	g.terminals = make(map[string]bool)
	g.nonTerminals = make(map[string]bool)
	g.chainProduction = make(map[string]string)
	g.First = make(map[string][]string)
	g.follow = make(map[string][]string)
	g.Last = make(map[string][]string)
	g.precede = make(map[string][]string)
	g.bigrammMatrix = make(map[string][]string)
	g.StartSymbol = ""

	g.GrammarParser(req.Productions)
	g.SearchChainRules()
	g.DeleteChainRules()

	var positiveTests []domain.TestCase
	var negativeTests []domain.TestCase
	generated := make(map[string]bool)
	attempts := 0

	fmt.Println("\n--- Грамматика после преобразования в ХНФ ---")
	err := g.ConvertToCNF()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	g.ComputeFirst()
	g.ComputeFollow()
	g.ComputeLast()
	g.ComputePrecede()
	g.BuildBigrammMatrix()
	g.PrintBigrammMatrix()

	for (len(positiveTests) < req.Pos || len(negativeTests) < req.Neg) && attempts < req.Attempts {
		word := g.GenerateRandomWord(req.MaxSteps, req.WProb)
		attempts++

		if _, exists := generated[word]; exists {
			continue
		}
		generated[word] = true

		isValid := g.ParseCYK(word)
		if isValid && len(positiveTests) < req.Pos {
			positiveTests = append(positiveTests, domain.TestCase{InLanguage: true, Word: word})
			g.Log.Infof("Тест %d: %s - Положительный", len(positiveTests), word)
		} else if !isValid && len(negativeTests) < req.Neg {
			negativeTests = append(negativeTests, domain.TestCase{InLanguage: false, Word: word})
			g.Log.Infof("Тест %d: %s - Отрицательный", len(negativeTests), word)
		}
	}

	allTests := append(positiveTests, negativeTests...)
	response := domain.TestResponse{Tests: allTests}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
