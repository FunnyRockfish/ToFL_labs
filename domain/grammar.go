package domain

import "go.uber.org/zap"

type Grammar struct {
	Gram            map[string][]OneProd
	NewProductions  map[string][][]string
	Terminals       map[string]bool
	NonTerminals    map[string]bool
	ChainProduction map[string]string
	First           map[string][]string
	Follow          map[string][]string
	Last            map[string][]string
	Precede         map[string][]string
	BigrammMatrix   map[string][]string
	StartSymbol     string
	Log             *zap.SugaredLogger
}

type OneProd struct {
	production []string
	isItChain  bool
}
