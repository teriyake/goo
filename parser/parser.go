package parser

import (
	"fmt"
	"strconv"
	"teriyake/goo/lexer"
)

type Identifier struct {
	Value string
}

type Number struct {
	Value float64
}

type Boolean struct {
	Value bool
}

type String struct {
	Value string
}

type Operator struct {
	Value string
}

type IfStatement struct {
	Condition interface{}
	ThenBlock interface{}
	ElseBlock interface{}
}

type Parser struct {
	lexer        *lexer.Lexer
	currentToken lexer.Token
	peekToken    lexer.Token
}

func NewParser(lexer *lexer.Lexer) *Parser {
	p := &Parser{lexer: lexer}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
	//fmt.Printf("nextToken - Current token: %s, Literal: %s\n", p.currentToken.Type, p.currentToken.Literal)
}

func (p *Parser) parseExpression() (interface{}, error) {
	//fmt.Printf("parseExpression - Start, Current token: %s, Literal: %s\n", p.currentToken.Type, p.currentToken.Literal)

	if p.currentToken.Type == lexer.ILLEGAL {
		return nil, fmt.Errorf("Unexpected token: %s", p.currentToken.Literal)
	}

	var result interface{}
	var err error
	switch p.currentToken.Type {
	case lexer.IDENT:
		if p.currentToken.Literal == "if" {
			result, err = p.parseIfStatement()
		} else {
			result = Identifier{Value: p.currentToken.Literal}
		}
	case lexer.NUMBER:
		literal := p.currentToken.Literal
		floatValue, err := strconv.ParseFloat(literal, 64)
		if err != nil {
			return nil, err
		}
		result = Number{Value: floatValue}
	case lexer.BOOL:
		if p.currentToken.Literal == "true" {
			result = Boolean{Value: true}
		} else {
			result = Boolean{Value: false}
		}
	case lexer.STRING:
		result = String{Value: p.currentToken.Literal}
	case lexer.OPERATOR:
		operator := Operator{Value: p.currentToken.Literal}
		p.nextToken()
		firstOperand, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		p.nextToken()
		secondOperand, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		result = []interface{}{operator, firstOperand, secondOperand}
	case lexer.LPAREN:
		p.nextToken()
		var nestedExpressions []interface{}
		for p.currentToken.Type != lexer.RPAREN && p.currentToken.Type != lexer.EOF {
			expr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			nestedExpressions = append(nestedExpressions, expr)
			if p.peekToken.Type == lexer.RPAREN {
				break
			}
			p.nextToken()
		}

		if !p.expectPeek(lexer.RPAREN) {
			return nil, fmt.Errorf("expected ')' after expression")
		}

		if len(nestedExpressions) == 1 {
			result = nestedExpressions[0]
		} else {
			result = nestedExpressions
		}

		p.nextToken()
	case lexer.RPAREN:
		p.nextToken()
		return nil, nil
	default:
		err = fmt.Errorf("Unexpected token: %s", p.currentToken.Literal)
	}

	//fmt.Printf("parseExpression - End, Parsed: %+v\n", result)
	return result, err
}

func (p *Parser) parseIfStatement() (IfStatement, error) {
	//fmt.Println("parseIfStatement - Start")
	var ifStmt IfStatement

	if !p.expectPeek(lexer.LPAREN) {
		return IfStatement{}, fmt.Errorf("expected '(' after 'if'")
	}

	p.nextToken()
	condition, err := p.parseExpression()
	if err != nil {
		return IfStatement{}, err
	}
	ifStmt.Condition = condition

	if !p.expectPeek(lexer.RPAREN) {
		return IfStatement{}, fmt.Errorf("expected ')' after if condition")
	}

	p.nextToken()
	thenBlock, err := p.parseExpression()
	if err != nil {
		return IfStatement{}, err
	}
	ifStmt.ThenBlock = []interface{}{thenBlock}

	if !p.expectPeek(lexer.RPAREN) {
		return IfStatement{}, fmt.Errorf("expected ')' at the end of the if statement")
	}

	//fmt.Println("parseIfStatement - End")
	return ifStmt, nil
}

func (p *Parser) expectPeek(t string) bool {
	if p.peekToken.Type == t || (t == lexer.RPAREN && p.peekToken.Type == lexer.EOF) {
		p.nextToken()
		return true
	} else {
		//fmt.Printf("expectPeek - Failed, Current token: %s, Expected: %s\n", p.peekToken.Type, t)
		return false
	}
}

func (p *Parser) peekTokenIs(t string) bool {
	return p.peekToken.Type == t
}

func (p *Parser) Parse() (interface{}, error) {
	//fmt.Println("Parse - Start")
	var ast []interface{}

	for p.currentToken.Type != lexer.EOF {
		expression, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		ast = append(ast, expression)

		//fmt.Printf("Parsed expression: %v\n", expression)
		p.nextToken()
	}

	//fmt.Println("Parse - End")
	return ast, nil
}
