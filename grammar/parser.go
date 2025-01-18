package grammar

func (g *Grammar) ParseCYK(word string) bool {
	n := len(word)
	if n == 0 {
		// Проверяем, существует ли правило S -> ε
		for _, prod := range g.gram[g.StartSymbol] {
			if len(prod.production) == 1 && prod.production[0] == "ε" {
				return true
			}
		}
		return false
	}

	// Таблица, тут каждый элемент - map[string]bool, хранящая нт, способные вывести подстроку от позиции i до j
	table := make([][]map[string]bool, n)
	for i := range table {
		table[i] = make([]map[string]bool, n)
		for j := range table[i] {
			table[i][j] = make(map[string]bool)
		}
	}

	// Заполнение таблицы для подстрок длины 1
	for i, char := range word {
		symbol := string(char)
		for lhs, prods := range g.gram {
			for _, prod := range prods {
				if len(prod.production) == 1 && prod.production[0] == symbol { //Если правило вида A -> a, где a — текущий символ слова, то в ячейке таблицы добавляется нетерминал A.
					table[i][i][lhs] = true
				}
				if len(prod.production) == 1 && prod.production[0] == "ε" && n == 0 { // может ли начальный вывести пустую строку?
					table[i][i][lhs] = true
				}
			}
		}
	}

	// Заполнение таблицы для подстрок длины >1
	for l := 2; l <= n; l++ {
		for i := 0; i <= n-l; i++ {
			j := i + l - 1
			for k := i; k < j; k++ {
				// Проходим по всем нетерминалам
				sortedNTs := g.sortedNonTerminals()
				for _, lhs := range sortedNTs {
					prods := g.gram[lhs]
					for _, prod := range prods {
						if len(prod.production) == 2 { // проверяем если A -> BC
							B := prod.production[0]
							C := prod.production[1]
							if table[i][k][B] && table[k+1][j][C] { // они выводят эти части подстроки?
								table[i][j][lhs] = true
							}
						}
					}
				}
			}
		}
	}

	// содержится ли начальный символ в ячейке, соответствующей всей строке table[0][n-1]?
	return table[0][n-1][g.StartSymbol]
}
