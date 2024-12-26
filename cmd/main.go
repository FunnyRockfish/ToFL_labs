package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"lab4_1/lexer"
	"lab4_1/logger"
	"lab4_1/parser"
)

// ANSI цветовые коды
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
)

func main() {
	log := logger.CreateLogger()
	testRegexes := []struct {
		regex string
		valid bool
	}{
		// Позитивные кейсы
		{"((abc)|ac(d)|(?2))|(?3)", true},
		{"(aa|bb)(?1)", true},
		{"(a|(bb))(a|(?2))", true},
		{"(a|(b|c))d", true},
		{"((a|b)c)*", true},
		{"(a)(?1)(a|(b|c))", true},
		{"(a*|(?:b|c))d", true},
		{"(a|b)(a|(bb(?4)))(a)", true},
		{"(a*|(?:b|c))d", true},

		{"(?=a)b", true},
		{"a(?=b|c)d", true},
		{"(a(?=b))c", true}, /*(?=a)b
		a(?=b|c)d*/

		// Негативные кейсы
		{"a|b)", false},
		{"((((a|b)))", false},
		{"(?3)(a|(b|c))", false},
		{"((a)(b)(c)(d)(e)(f)(g)(h)(i)(j))", false},
		{"(a)(?2)", false},
		{"(?=(a))", false},
		{"(?=a(?=b))", false},
		{"(?:aab)(?1)", false},
		{"((abbb)|(baaa))(?2)(?1)\\2", false},
	}

	for _, tr := range testRegexes {
		fmt.Printf("Регулярное выражение: %s\n", tr.regex)

		if !strings.Contains(tr.regex, "?=") {
			preparedRegex := replaceElems(tr.regex)
			isRegexValid := checkRegexWithGo(preparedRegex)
			if !isRegexValid {
				log.Error("НЕКОРРЕКТНЫЙ СИНТАКСИС РЕГЕКСА! ЧТО ТО ГДЕ ТО НЕ ТО НАПИСАНО")
				continue
			}
		}

		lex := lexer.NewLexer(tr.regex)
		pars := parser.NewParser(lex)

		ast, err := pars.Parse()

		if err != nil {
			// Вывод ошибки красным
			fmt.Printf("%sОшибка при парсинге: %s%s\n", Red, err.Error(), Reset)
			continue
		}

		fmt.Println("Построенное AST:")
		parser.PrintAST(ast, "")

		validator := parser.NewValidator(log)
		validator.Validate(ast)

		// Проверка наличия ошибок в валидаторе
		if len(validator.Errors()) > 0 && !tr.valid {
			fmt.Printf("%sТест успешно пройден.%s\n", Green, Reset)
		} else if len(validator.Errors()) > 0 && tr.valid {
			fmt.Printf("%sТест провален.%s\n", Red, Reset)
		} else if len(validator.Errors()) == 0 && tr.valid {
			fmt.Printf("%sТест успешно пройден.%s\n", Green, Reset)
		} else {
			fmt.Printf("%sТест провален.%s\n", Red, Reset)
		}

		fmt.Println()

		file, err := os.Create("ast.dot")
		if err != nil {
			fmt.Printf("%sОшибка при создании файла: %s%s\n", Red, err.Error(), Reset)
			return
		}
		defer file.Close()

		parser.PrintASTDot(ast, file)
	}
}

func checkRegexWithGo(regex string) bool {
	fmt.Println("Check for: ", regex)
	_, err := regexp.Compile(regex)
	return err == nil
}

func replaceElems(regex string) string {
	pattern := regexp.MustCompile(`\\[1-9]|\?[1-9]|\?:`)

	replaced := pattern.ReplaceAllStringFunc(regex, func(match string) string {
		if strings.HasPrefix(match, `\`) {
			return fmt.Sprint("a", match[2:])
		} else if strings.HasPrefix(match, `?`) {
			return fmt.Sprint("b", match[2:])
		} else if strings.HasPrefix(match, `?:`) {
			return fmt.Sprint("c", match[2:])
		}
		return match
	})

	return replaced
}
