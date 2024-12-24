package main

import (
	"fmt"
	"os"

	"lab4_1/lexer"
	"lab4_1/logger"
	"lab4_1/parser"
)

func main() {
	log := logger.CreateLogger()
	testRegexes := []struct {
		regex string
		valid bool
	}{
		{"(a(?1)b|c)", false},
	}

	for _, tr := range testRegexes {
		fmt.Printf("Регулярное выражение: %s\n", tr.regex)
		lex := lexer.NewLexer(tr.regex)
		pars := parser.NewParser(lex)

		ast, err := pars.Parse()
		if err != nil {
			fmt.Printf("Ошибка парсинга: %v\n\n", err)
			continue
		}

		fmt.Println("Построенное AST:")
		parser.PrintAST(ast, "")

		validator := parser.NewValidator(9, log)
		validator.Validate(ast)

		fmt.Println()

		file, err := os.Create("ast.dot")
		if err != nil {
			fmt.Println("Ошибка при создании файла:", err)
			return
		}
		defer file.Close()

		parser.PrintASTDot(ast, file)
	}
}
