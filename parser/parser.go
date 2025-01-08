package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"lab4_1/lexer"
)

type ASTNode interface {
}

type Char struct {
	value rune
}

type Concat struct {
	left  ASTNode
	right ASTNode
}

type Union struct {
	left  ASTNode
	right ASTNode
}

type Star struct {
	node ASTNode
}

type Group struct {
	node        ASTNode
	capturing   bool
	groupNumber int
}

type BackReference struct {
	GroupNumber int
}

type LookAhead struct {
	node ASTNode
}

type SubPatternReference struct {
	GroupNumber int
}

type Parser struct {
	lexer        *lexer.Lexer
	currentToken lexer.Token
	peekToken    lexer.Token

	groupCount int
	maxGroups  int
}

func NewParser(lexer *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:     lexer,
		maxGroups: 9,
	}

	p.NextToken()
	p.NextToken()
	return p
}

func (p *Parser) NextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) Parse() (ASTNode, error) {
	node, err := p.parseRG()
	if err != nil {
		return nil, err
	}
	if p.currentToken.Type != lexer.TOKEN_EOF {
		return nil, fmt.Errorf("ожидался конец ввода, но найден токен '%s'", p.currentToken.Value)
	}

	return node, nil
}

func (p *Parser) parseTerm() (ASTNode, error) {
	var node ASTNode
	var err error

	switch p.currentToken.Type {
	case lexer.TOKEN_CHAR:
		node = &Char{value: rune(p.currentToken.Value[0])}
		p.NextToken()
	case lexer.TOKEN_LPAREN, lexer.TOKEN_NON_CAPTURING_GROUP_START, lexer.TOKEN_POSITIVE_LOOKAHEAD_START:
		node, err = p.parseGroup()
		if err != nil {
			return nil, err
		}
	case lexer.TOKEN_BACKREFERENCE:
		num, err := strconv.Atoi(p.currentToken.Value[1:])
		if err != nil {
			return nil, fmt.Errorf("недопустимая ссылка на группу: %s", p.currentToken.Value)
		}
		node = &BackReference{GroupNumber: num}
		p.NextToken()
	case lexer.TOKEN_SUBPATTERN_REFERENCE:
		num, err := strconv.Atoi(p.currentToken.Value)
		if err != nil {
			return nil, fmt.Errorf("недопустимая ссылка на подпаттерн: %s", p.currentToken.Value)
		}
		node = &SubPatternReference{GroupNumber: num}
		p.NextToken()
	case lexer.TOKEN_ILLEGAL:
		return nil, fmt.Errorf("обнаружен недопустимый символ: '%s'", p.currentToken.Value)

	default:
		return nil, nil
	}

	if p.currentToken.Type == lexer.TOKEN_STAR {
		node = &Star{node: node}
		p.NextToken()
	}

	return node, nil
}

func (p *Parser) parseGroup() (ASTNode, error) {
	switch p.currentToken.Type {
	case lexer.TOKEN_LPAREN:
		p.NextToken()
		p.groupCount++
		if p.groupCount > p.maxGroups {
			return nil, fmt.Errorf("превышено максимальное количество групп захвата (%d)", p.maxGroups)
		}
		groupNum := p.groupCount
		node, err := p.parseRG()
		if err != nil {
			return nil, err
		}
		if p.currentToken.Type != lexer.TOKEN_RPAREN {
			return nil, fmt.Errorf("ожидалась закрывающая скобка ')', но найден '%s'", p.currentToken.Value)
		}
		p.NextToken()
		return &Group{
			node:        node,
			capturing:   true,
			groupNumber: groupNum,
		}, nil
	case lexer.TOKEN_NON_CAPTURING_GROUP_START:
		p.NextToken()
		node, err := p.parseRG()
		if err != nil {
			return nil, err
		}
		if p.currentToken.Type != lexer.TOKEN_RPAREN {
			return nil, fmt.Errorf("ожидалась закрывающая скобка ')', но найден '%s'", p.currentToken.Value)
		}
		p.NextToken()
		return &Group{
			node:      node,
			capturing: false,
		}, nil
	case lexer.TOKEN_POSITIVE_LOOKAHEAD_START:
		p.NextToken()
		node, err := p.parseRG()
		if err != nil {
			return nil, fmt.Errorf("ожидалась закрывающая скобка ')', но найден '%s'", p.currentToken.Value)
		}
		p.NextToken()
		return &LookAhead{
			node: node,
		}, nil
	default:
		return nil, fmt.Errorf("неожиданный токен '%s' при разборе группы", p.currentToken.Value)
	}
}

func (p *Parser) parseConcat() (ASTNode, error) {
	nodes := []ASTNode{}

	for {
		node, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		if node == nil {
			break
		}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	concat := nodes[0]
	for i := 1; i < len(nodes); i++ {
		concat = &Concat{
			left:  concat,
			right: nodes[i],
		}
	}

	return concat, nil
}

func (p *Parser) parseRG() (ASTNode, error) {
	node, err := p.parseConcat()
	if err != nil {
		return nil, err
	}

	for p.currentToken.Type == lexer.TOKEN_PIPE {
		p.NextToken()
		right, err := p.parseConcat()
		if err != nil {
			return nil, err
		}
		node = &Union{
			left:  node,
			right: right,
		}
	}
	return node, nil
}

func PrintASTDot(node ASTNode, writer io.Writer) {
	fmt.Fprintln(writer, "digraph AST {")
	fmt.Fprintln(writer, "    node [shape=box];")

	nodeID := 0
	nodeMap := make(map[ASTNode]int)

	var traverse func(n ASTNode) int
	traverse = func(n ASTNode) int {
		if id, exists := nodeMap[n]; exists {
			return id
		}

		currentID := nodeID
		nodeMap[n] = currentID
		nodeID++

		var label string
		switch n := n.(type) {
		case *Char:
			label = fmt.Sprintf("Char: '%c'", n.value)
		case *Concat:
			label = "Concat"
		case *Union:
			label = "Union"
		case *Star:
			label = "Star"
		case *Group:
			if n.capturing {
				label = fmt.Sprintf("Group (Capturing, #%d)", n.groupNumber)
			} else {
				label = "Group (Non-Capturing)"
			}
		case *BackReference:
			label = fmt.Sprintf("BackReference: #%d", n.GroupNumber)
		case *LookAhead:
			label = "LookAhead"
		case *SubPatternReference:
			label = fmt.Sprintf("SubPatternReference: #%d", n.GroupNumber)
		default:
			label = "Unknown"
		}

		label = escapeString(label)

		fmt.Fprintf(writer, "    node%d [label=\"%s\"];\n", currentID, label)

		switch n := n.(type) {
		case *Concat:
			leftID := traverse(n.left)
			rightID := traverse(n.right)
			fmt.Fprintf(writer, "    node%d -> node%d;\n", currentID, leftID)
			fmt.Fprintf(writer, "    node%d -> node%d;\n", currentID, rightID)
		case *Union:
			leftID := traverse(n.left)
			rightID := traverse(n.right)
			fmt.Fprintf(writer, "    node%d -> node%d;\n", currentID, leftID)
			fmt.Fprintf(writer, "    node%d -> node%d;\n", currentID, rightID)
		case *Star:
			childID := traverse(n.node)
			fmt.Fprintf(writer, "    node%d -> node%d;\n", currentID, childID)
		case *Group:
			childID := traverse(n.node)
			fmt.Fprintf(writer, "    node%d -> node%d;\n", currentID, childID)
		case *BackReference:
			// BackReference не имеет дочерних узлов
		case *LookAhead:
			childID := traverse(n.node)
			fmt.Fprintf(writer, "    node%d -> node%d;\n", currentID, childID)
		case *Char:
		case *SubPatternReference:

		default:
		}

		return currentID
	}

	traverse(node)
	fmt.Fprintln(writer, "}")
}

func escapeString(s string) string {
	s = strconv.Quote(s)
	return s[1 : len(s)-1]
}

func PrintAST(node ASTNode, indent string) {
	switch n := node.(type) {
	case *Char:
		fmt.Printf("%sChar: '%c'\n", indent, n.value)
	case *Concat:
		fmt.Printf("%sConcat:\n", indent)
		PrintAST(n.left, indent+"  ")
		PrintAST(n.right, indent+"  ")
	case *Union:
		fmt.Printf("%sUnion:\n", indent)
		PrintAST(n.left, indent+"  ")
		PrintAST(n.right, indent+"  ")
	case *Star:
		fmt.Printf("%sStar:\n", indent)
		PrintAST(n.node, indent+"  ")
	case *Group:
		if n.capturing {
			fmt.Printf("%sGroup (Capturing, #%d):\n", indent, n.groupNumber)
		} else {
			fmt.Printf("%sGroup (Non-Capturing):\n", indent)
		}
		PrintAST(n.node, indent+"  ")
	case *BackReference:
		fmt.Printf("%sBackReference: #%d\n", indent, n.GroupNumber)
	case *LookAhead:
		fmt.Printf("%sLookAhead:\n", indent)
		PrintAST(n.node, indent+"  ")
	case *SubPatternReference:
		fmt.Printf("%sSubPatternReference: #%d\n", indent, n.GroupNumber) // Исправлено
	default:
		fmt.Printf("%sUnknown node type\n", indent)
	}
}

// ------------------ВАЛИДАТОР----------------//

func (v *Validator) Errors() []string {
	return v.errors
}

type Validator struct {
	log    *zap.SugaredLogger
	errors []string

	maxGroups int

	groupsByNumber map[int]*Group

	checkingSubPattern map[int]bool
}

func NewValidator(log *zap.SugaredLogger) *Validator {
	return &Validator{
		log:    log,
		errors: []string{},

		maxGroups:          9,
		groupsByNumber:     make(map[int]*Group),
		checkingSubPattern: make(map[int]bool),
	}
}

func (v *Validator) Validate(root ASTNode) (bool, []string) {
	v.collectAllGroups(root)
	v.checkLookAheadConstraints(root)

	inSet := make(map[int]bool)
	_, err := v.computeDefGroups(root, inSet)
	if err != nil {
		v.errors = append(v.errors, err.Error())
	}

	if len(v.errors) == 0 {
		v.log.Infof("Регулярное выражение корректно.")
		return true, nil
	} else {
		v.log.Errorf("Ошибки в регулярном выражении:")
		for _, e := range v.errors {
			v.log.Errorf("- %s", e)
		}
		return false, v.errors
	}
}

func (v *Validator) collectAllGroups(node ASTNode) {
	switch n := node.(type) {
	case *Concat:
		v.collectAllGroups(n.left)
		v.collectAllGroups(n.right)

	case *Union:
		v.collectAllGroups(n.left)
		v.collectAllGroups(n.right)

	case *Star:
		v.collectAllGroups(n.node)

	case *Group:
		if n.capturing {
			if n.groupNumber > v.maxGroups {
				v.errors = append(v.errors,
					fmt.Sprintf("Превышено максимальное количество групп (>%d)", v.maxGroups))
			}
			v.groupsByNumber[n.groupNumber] = n
		}
		v.collectAllGroups(n.node)

	case *LookAhead:
		v.collectAllGroups(n.node)
	}
}

func (v *Validator) checkLookAheadConstraints(node ASTNode) {
	switch n := node.(type) {
	case *Concat:
		v.checkLookAheadConstraints(n.left)
		v.checkLookAheadConstraints(n.right)

	case *Union:
		v.checkLookAheadConstraints(n.left)
		v.checkLookAheadConstraints(n.right)

	case *Star:
		v.checkLookAheadConstraints(n.node)

	case *Group:
		v.checkLookAheadConstraints(n.node)

	case *LookAhead:
		if v.containsCapturingGroup(n.node) {
			v.errors = append(v.errors,
				"Запрещено использовать группы захвата внутри опережающей проверки (LookAhead)")
		}
		if v.containsLookAhead(n.node) {
			v.errors = append(v.errors,
				"Запрещено использовать вложенные опережающие проверки (LookAhead внутри LookAhead)")
		}
		v.checkLookAheadConstraints(n.node)

	default:
	}
}

func (v *Validator) containsCapturingGroup(node ASTNode) bool {
	switch n := node.(type) {
	case *Group:
		if n.capturing {
			return true
		}
		return v.containsCapturingGroup(n.node)
	case *Concat:
		return v.containsCapturingGroup(n.left) || v.containsCapturingGroup(n.right)
	case *Union:
		return v.containsCapturingGroup(n.left) || v.containsCapturingGroup(n.right)
	case *Star:
		return v.containsCapturingGroup(n.node)
	case *LookAhead:
		return v.containsCapturingGroup(n.node)
	default:
		return false
	}
}

func (v *Validator) containsLookAhead(node ASTNode) bool {
	switch n := node.(type) {
	case *LookAhead:
		return true
	case *Group:
		return v.containsLookAhead(n.node)
	case *Concat:
		return v.containsLookAhead(n.left) || v.containsLookAhead(n.right)
	case *Union:
		return v.containsLookAhead(n.left) || v.containsLookAhead(n.right)
	case *Star:
		return v.containsLookAhead(n.node)
	default:
		return false
	}
}

func (v *Validator) computeDefGroups(node ASTNode, inSet map[int]bool) (map[int]bool, error) {
	switch n := node.(type) {

	case *Char:
		return copySet(inSet), nil

	case *Concat:
		leftOut, err := v.computeDefGroups(n.left, copySet(inSet))
		if err != nil {
			return nil, err
		}
		rightOut, err := v.computeDefGroups(n.right, leftOut)
		if err != nil {
			return nil, err
		}
		return rightOut, nil

	case *Union:
		leftOut, err := v.computeDefGroups(n.left, copySet(inSet))
		if err != nil {
			return nil, err
		}
		rightOut, err := v.computeDefGroups(n.right, copySet(inSet))
		if err != nil {
			return nil, err
		}
		return intersectSets(leftOut, rightOut), nil

	case *Star:
		childOut, err := v.computeDefGroups(n.node, copySet(inSet))
		if err != nil {
			return nil, err
		}
		return intersectSets(inSet, childOut), nil

	case *Group:
		if n.capturing {
			childOut, err := v.computeDefGroups(n.node, copySet(inSet))
			if err != nil {
				return nil, err
			}
			childOut[n.groupNumber] = true
			return childOut, nil
		} else {
			return v.computeDefGroups(n.node, inSet)
		}

	case *LookAhead:
		_, err := v.computeDefGroups(n.node, copySet(inSet))
		return copySet(inSet), err

	case *BackReference:
		if !inSet[n.GroupNumber] {
			return nil, fmt.Errorf("ссылка на неинициализированную группу \\%d", n.GroupNumber)
		}
		return copySet(inSet), nil

	case *SubPatternReference:
		grp := v.groupsByNumber[n.GroupNumber]
		if grp == nil {
			return nil, fmt.Errorf("ссылка на несуществующую группу (?%d)", n.GroupNumber)
		}
		if v.checkingSubPattern[n.GroupNumber] {
		} else {
			v.checkingSubPattern[n.GroupNumber] = true
			_, err := v.computeDefGroups(grp.node, copySet(inSet))
			v.checkingSubPattern[n.GroupNumber] = false
			if err != nil {
				return nil, err
			}
		}
		return copySet(inSet), nil

	default:
		return nil, fmt.Errorf("неизвестный тип узла при валидации")
	}
}

func copySet(m map[int]bool) map[int]bool {
	res := make(map[int]bool, len(m))
	for k := range m {
		res[k] = true
	}
	return res
}

func intersectSets(a, b map[int]bool) map[int]bool {
	res := make(map[int]bool)
	for k := range a {
		if b[k] {
			res[k] = true
		}
	}
	return res
}

type ContextFreeGrammar struct {
	Grammar          map[string][]string
	unionCounter     int
	concatCounter    int
	starCounter      int
	lookAheadCounter int
	charCounter      int
	charCache        map[rune]string
	firstSymbol      string
	Order            []string
}

func NewCFGBuilder() *ContextFreeGrammar {
	return &ContextFreeGrammar{
		Grammar:          map[string][]string{},
		unionCounter:     0,
		concatCounter:    0,
		starCounter:      0,
		lookAheadCounter: 0,
		charCounter:      0,
		charCache:        make(map[rune]string),
		firstSymbol:      "",
		Order:            make([]string, 0),
	}
}

func (cfg *ContextFreeGrammar) BuildCFGbyAST(node ASTNode) string {
	switch n := node.(type) {
	case *Union:
		nonTerminal := "U" + strconv.Itoa(cfg.unionCounter)
		cfg.unionCounter++

		if cfg.firstSymbol == "" {
			cfg.firstSymbol = nonTerminal
			cfg.Grammar["S"] = append(cfg.Grammar["S"], nonTerminal)
			cfg.Order = append(cfg.Order, "S")
		}

		if nonTerminal != "S" && !contains(cfg.Order, nonTerminal) {
			cfg.Order = append(cfg.Order, nonTerminal)
		}

		leftSymbol := cfg.BuildCFGbyAST(n.left)
		rightSymbol := cfg.BuildCFGbyAST(n.right)
		cfg.Grammar[nonTerminal] = append(cfg.Grammar[nonTerminal], leftSymbol)
		cfg.Grammar[nonTerminal] = append(cfg.Grammar[nonTerminal], rightSymbol)

		return nonTerminal

	case *Group:
		nonTerminal := "G" + strconv.Itoa(n.groupNumber)

		if cfg.firstSymbol == "" {
			cfg.firstSymbol = nonTerminal
			cfg.Grammar["S"] = append(cfg.Grammar["S"], nonTerminal)
			cfg.Order = append(cfg.Order, "S")
		}

		if nonTerminal != "S" && !contains(cfg.Order, nonTerminal) {
			cfg.Order = append(cfg.Order, nonTerminal)
		}

		if _, exists := cfg.Grammar[nonTerminal]; !exists {
			groupBody := cfg.BuildCFGbyAST(n.node)
			cfg.Grammar[nonTerminal] = append(cfg.Grammar[nonTerminal], groupBody)
		}

		return nonTerminal

	case *Concat:
		nonTerminal := "C" + strconv.Itoa(cfg.concatCounter)
		cfg.concatCounter++

		if cfg.firstSymbol == "" {
			cfg.firstSymbol = nonTerminal
			cfg.Grammar["S"] = append(cfg.Grammar["S"], nonTerminal)
			cfg.Order = append(cfg.Order, "S")
		}

		if nonTerminal != "S" && !contains(cfg.Order, nonTerminal) {
			cfg.Order = append(cfg.Order, nonTerminal)
		}

		leftSymbol := cfg.BuildCFGbyAST(n.left)
		rightSymbol := cfg.BuildCFGbyAST(n.right)

		concatenation := leftSymbol + " " + rightSymbol
		cfg.Grammar[nonTerminal] = append(cfg.Grammar[nonTerminal], concatenation)

		return nonTerminal

	case *Char:
		sym, exists := cfg.charCache[n.value]
		if exists {
			return sym
		}

		nonTerminal := strings.ToUpper(string(n.value))

		if cfg.firstSymbol == "" {
			cfg.firstSymbol = nonTerminal
			cfg.Grammar["S"] = append(cfg.Grammar["S"], nonTerminal)
			cfg.Order = append(cfg.Order, "S") // Добавляем S первым
		}

		if nonTerminal != "S" && !contains(cfg.Order, nonTerminal) {
			cfg.Order = append(cfg.Order, nonTerminal)
		}

		cfg.Grammar[nonTerminal] = append(cfg.Grammar[nonTerminal], string(n.value))

		cfg.charCache[n.value] = nonTerminal

		return nonTerminal

	case *Star:
		nonTerminal := "S" + strconv.Itoa(cfg.starCounter)
		cfg.starCounter++

		if cfg.firstSymbol == "" {
			cfg.firstSymbol = nonTerminal
			cfg.Grammar["S"] = append(cfg.Grammar["S"], nonTerminal)
			cfg.Order = append(cfg.Order, "S")
		}

		if nonTerminal != "S" && !contains(cfg.Order, nonTerminal) {
			cfg.Order = append(cfg.Order, nonTerminal)
		}
		childSymbol := cfg.BuildCFGbyAST(n.node)

		cfg.Grammar[nonTerminal] = append(cfg.Grammar[nonTerminal], "ε")                         // S -> ε
		cfg.Grammar[nonTerminal] = append(cfg.Grammar[nonTerminal], nonTerminal+" "+childSymbol) // S -> S X

		return nonTerminal

	case *SubPatternReference:
		nonTerminal := "G" + strconv.Itoa(n.GroupNumber)

		if cfg.firstSymbol == "" {
			cfg.firstSymbol = nonTerminal
			cfg.Grammar["S"] = append(cfg.Grammar["S"], nonTerminal)
			cfg.Order = append(cfg.Order, "S")
		}

		if nonTerminal != "S" && !contains(cfg.Order, nonTerminal) {
			cfg.Order = append(cfg.Order, nonTerminal)
		}

		return nonTerminal

	case *BackReference:
		nonTerminal := "G" + strconv.Itoa(n.GroupNumber)

		if cfg.firstSymbol == "" {
			cfg.firstSymbol = nonTerminal
			cfg.Grammar["S"] = append(cfg.Grammar["S"], nonTerminal)
			cfg.Order = append(cfg.Order, "S") // Добавляем S первым
		}

		if nonTerminal != "S" && !contains(cfg.Order, nonTerminal) {
			cfg.Order = append(cfg.Order, nonTerminal)
		}

		return nonTerminal

	case *LookAhead:
		fmt.Println("here")

		nonTerminal := "L" + strconv.Itoa(cfg.lookAheadCounter)
		cfg.lookAheadCounter++

		if cfg.firstSymbol == "" {
			cfg.firstSymbol = nonTerminal
			cfg.Grammar["S"] = append(cfg.Grammar["S"], nonTerminal)
			cfg.Order = append(cfg.Order, "S") // Добавляем S первым
		}

		if nonTerminal != "S" && !contains(cfg.Order, nonTerminal) {
			cfg.Order = append(cfg.Order, nonTerminal)
		}

		cfg.Grammar[nonTerminal] = append(cfg.Grammar[nonTerminal], "ε")
		return nonTerminal

	default:
		fmt.Printf("Неизвестный тип узла: %T\n", n)
		return ""
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (cfg *ContextFreeGrammar) PrintCFG() []string {
	grammar := make([]string, 0)
	fmt.Println()
	for _, nt := range cfg.Order {
		terms, exists := cfg.Grammar[nt]
		if !exists {
			continue
		}
		production := nt + " -> "
		for _, term := range terms {
			production += term + " | "
		}
		production = strings.TrimSuffix(production, " | ")
		grammar = append(grammar, production)
		fmt.Println(production)
	}
	return grammar
}
