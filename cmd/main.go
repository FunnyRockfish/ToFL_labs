package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"lab4_1/domain"
	"lab4_1/lexer"
	"lab4_1/logger"
	"lab4_1/parser"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
)

func main() {
	handleRegex()
	r := mux.NewRouter()
	r.HandleFunc("/buildGrammar", corsMiddleware(doSkeletonGrammar)).Methods(http.MethodPost, http.MethodOptions)

	port := "8010"
	fmt.Printf("Сервер запущен на http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		fmt.Printf("Ошибка запуска сервера: %s\n", err.Error())
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func handleRegex() {
	log := logger.CreateLogger()

	rg := "(?=aaa)a*"
	lex := lexer.NewLexer(rg)
	pars := parser.NewParser(lex)

	ast, err := pars.Parse()

	validator := parser.NewValidator(log)
	isCorrect, errors := validator.Validate(ast)
	fmt.Println(errors)
	fmt.Println(isCorrect)

	fileName := "ast.dot"
	file, err := os.Create(fileName)
	if err != nil {
		log.Error(err)
		return
	}
	defer file.Close()
	parser.PrintASTDot(ast, file)

	cfgBuilder := parser.NewCFGBuilder()
	cfgBuilder.BuildCFGbyAST(ast)
	grammar := cfgBuilder.PrintCFG()
	fmt.Println(grammar)
}

func doSkeletonGrammar(w http.ResponseWriter, r *http.Request) {
	log := logger.CreateLogger()

	var regexReq domain.RegexReq
	err := json.NewDecoder(r.Body).Decode(&regexReq)
	if err != nil {
		http.Error(w, "Ошибка декодирования JSON", http.StatusBadRequest)
		return
	}

	rg := regexReq.Rg
	lex := lexer.NewLexer(rg)
	pars := parser.NewParser(lex)

	ast, err := pars.Parse()
	if err != nil {
		errs := make([]string, 1)
		errs[0] = err.Error()
		response := domain.RegexResponse{
			IsCorrect: false,
			Errors:    errs,
			AST:       nil,
			Grammar:   nil,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	validator := parser.NewValidator(log)
	isCorrect, errors := validator.Validate(ast)

	fileName := "ast.dot"
	file, err := os.Create(fileName)
	if err != nil {
		http.Error(w, "Ошибка при создании файла AST", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	parser.PrintASTDot(ast, file)

	fileContent, err := ioutil.ReadFile(fileName)
	if err != nil {
		http.Error(w, "Ошибка при чтении файла AST", http.StatusInternalServerError)
		return
	}
	dotString := string(fileContent)
	dotLines := strings.Split(dotString, "\n")

	cfgBuilder := parser.NewCFGBuilder()
	cfgBuilder.BuildCFGbyAST(ast)
	grammar := cfgBuilder.PrintCFG()

	response := domain.RegexResponse{
		IsCorrect: isCorrect,
		Errors:    errors,
		AST:       dotLines,
		Grammar:   grammar,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
