package lexer

import (
	"fmt"
	"unicode"

	"go.uber.org/zap"
	"lab4_1/logger"
)

type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_PIPE
	TOKEN_STAR
	TOKEN_NON_CAPTURING_GROUP_START
	TOKEN_POSITIVE_LOOKAHEAD_START
	TOKEN_BACKREFERENCE
	TOKEN_CHAR
	TOKEN_SUBPATTERN_REFERENCE
	TOKEN_ILLEGAL
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           rune
	logger       *zap.SugaredLogger
}

func NewLexer(input string) *Lexer {
	l := Lexer{
		input:  input,
		logger: logger.CreateLogger(),
	}
	l.readChar()
	return &l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = rune(l.input[l.readPosition])
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	tok := Token{}

	switch {
	case l.ch == 0:
		tok.Type = TOKEN_EOF
		tok.Value = ""

	case l.ch == '(':
		if l.peekString("?:") {
			l.readChar() // (
			l.readChar() // ?
			l.readChar() // :
			tok.Type = TOKEN_NON_CAPTURING_GROUP_START
			tok.Value = "(?:"
		} else if l.peekString("?=") {
			l.readChar() // (
			l.readChar() // ?
			l.readChar() // =
			tok.Type = TOKEN_POSITIVE_LOOKAHEAD_START
			tok.Value = "?="
		} else if l.peekString("?") && l.peekNextIsDigitAfter(1) {
			l.readChar()
			l.readChar()
			num := l.ch
			if !unicode.IsDigit(l.ch) || l.ch == '0' {
				l.logger.Error("Неправильный номер группы")
				tok = Token{Type: TOKEN_EOF, Value: ""}
			} else {
				tok.Type = TOKEN_SUBPATTERN_REFERENCE
				tok.Value = fmt.Sprintf("%c", num)
				l.readChar()
				if l.ch == ')' {
					l.readChar() // )
				} else {
					l.logger.Error("Ожидалась закрывающая скобка ')' после ссылки на подпаттерн")
				}
			}
		} else {
			tok.Type = TOKEN_LPAREN
			tok.Value = "("
			l.readChar()
		}
	case l.ch == ')':
		tok.Type = TOKEN_RPAREN
		tok.Value = ")"
		l.readChar()
	case l.ch == '|':
		tok.Type = TOKEN_PIPE
		tok.Value = "|"
		l.readChar()
	case l.ch == '*':
		tok.Type = TOKEN_STAR
		tok.Value = "*"
		l.readChar()
	case l.ch == '\\':
		l.readChar()
		if unicode.IsDigit(l.ch) && l.ch != '0' {
			tok.Type = TOKEN_BACKREFERENCE
			tok.Value = "\\" + string(l.ch)
			l.readChar()
		} else {
			l.logger.Error("Посторонний символ после слэша! Символ:", l.ch)
			l.readChar()
		}
	case unicode.IsLower(l.ch):
		tok.Type = TOKEN_CHAR
		tok.Value = string(l.ch)
		l.readChar()
	default:
		tok.Type = TOKEN_ILLEGAL
		tok.Value = string(l.ch)
		l.logger.Error("Неподдерживаемый символ:", l.ch)
		l.readChar()

	}

	return tok
}

func (l *Lexer) peekNextIsDigitAfter(offset int) bool {
	pos := l.readPosition + offset
	if pos >= len(l.input) {
		return false
	}
	nextChar := rune(l.input[pos])
	return unicode.IsDigit(nextChar) && nextChar != '0'
}

func (l *Lexer) peekNextIsDigit() bool {
	if l.readPosition >= len(l.input) {
		return false
	}
	nextChar := rune(l.input[l.readPosition])
	return unicode.IsDigit(nextChar) && nextChar != '0'
}

func (l *Lexer) skipWhitespace() {
	for unicode.IsSpace(l.ch) {
		l.readChar()
	}
}

func (l *Lexer) peekString(s string) bool {
	if l.readPosition+len(s)-1 >= len(l.input) {
		return false
	}
	return l.input[l.readPosition:l.readPosition+len(s)] == s
}
