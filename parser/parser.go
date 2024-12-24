package parser

import (
	"fmt"
	"io"
	"strconv"

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

type Validator struct {
	maxGroups     int
	currentGroups []int
	definedGroups map[int]bool
	errors        []string
	log           *zap.SugaredLogger
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
		num, err := strconv.Atoi(p.currentToken.Value) // Изменено на p.currentToken.Value без "(?"
		if err != nil {
			return nil, fmt.Errorf("недопустимая ссылка на подпаттерн: %s", p.currentToken.Value)
		}
		node = &SubPatternReference{GroupNumber: num}
		p.NextToken()

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

	// Рекурсивная функция для обхода AST и создания узлов и связей
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

		// Экранирование кавычек в метках
		label = escapeString(label)

		fmt.Fprintf(writer, "    node%d [label=\"%s\"];\n", currentID, label)

		// Создание связей между узлами
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

// экранирует кавычки и другие специальные символы в строке
func escapeString(s string) string {
	s = strconv.Quote(s)
	// Убираем начальные и конечные кавычки, добавленные strconv.Quote
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

func NewValidator(maxGroups int, log *zap.SugaredLogger) *Validator {
	return &Validator{
		errors:        make([]string, 0),
		maxGroups:     maxGroups,
		log:           log,
		definedGroups: map[int]bool{},
	}
}

func (v *Validator) Validate(node ASTNode) {
	v.visit(node)

	if len(v.errors) > 0 {
		v.log.Error("Ошибки в регулярном выражении: ")
		for _, e := range v.errors {
			v.log.Errorf("- %s\n", e)
		}
	} else {
		v.log.Info("Регулярное выражение корректно.")
	}
}

func (v *Validator) visit(node ASTNode) {
	switch n := node.(type) {
	case *Char:
	// ничего
	case *Concat:
		v.visit(n.left)
		v.visit(n.right)
	case *Star:
		v.visit(n.node)
	case *Union:
		v.visit(n.left)
		v.visit(n.right)
	case *Group:
		if n.capturing {
			if n.groupNumber > v.maxGroups {
				v.errors = append(v.errors, fmt.Sprintf("Превышено максимальное количество групп захвата (%d)", v.maxGroups))
			}
			v.definedGroups[n.groupNumber] = true
			v.visit(n.node)
		} else {
			v.visit(n.node)
		}
	case *BackReference:
		if !v.definedGroups[n.GroupNumber] {
			v.errors = append(v.errors, fmt.Sprintf("Ссылка на неинициализированную группу \\%d", n.GroupNumber))
		}
	case *LookAhead:
		if v.containsGroup(n.node) {
			v.errors = append(v.errors, "Группы захвата не разрешены внутри опережающих проверок")
		}
		if v.containsLookAhead(n.node) {
			v.errors = append(v.errors, "Опережающие проверки не могут содержать другие опережающие проверки")
		}
		v.visit(n.node)
	case *SubPatternReference:
		if !v.definedGroups[n.GroupNumber] {
			v.errors = append(v.errors, fmt.Sprintf("Ссылка на неинициализированный подпаттерн (?%d)", n.GroupNumber))
		}
	}
}

func (v *Validator) containsGroup(node ASTNode) bool {
	switch n := node.(type) {
	case *Group:
		if n.capturing {
			return true
		}
	case *LookAhead:
		return true
	case *Concat:
		return v.containsGroup(n.left) || v.containsGroup(n.right)
	case *Union:
		return v.containsGroup(n.left) || v.containsGroup(n.right)
	case *Star:
		return v.containsGroup(n.node)
	default:
		return false
	}
	return false
}

func (v *Validator) containsLookAhead(node ASTNode) bool {
	switch node.(type) {
	case *Group:
		return v.containsLookAhead(node.(*Group).node)
	case *LookAhead:
		return true
	case *Concat:
		return v.containsLookAhead(node.(*Concat).right) || v.containsLookAhead(node.(*Concat).left)
	case *Union:
		return v.containsLookAhead(node.(*Union).right) || v.containsLookAhead(node.(*Union).left)
	case *Star:
		return v.containsLookAhead(node.(*Star).node)
	default:
		return false
	}
}
