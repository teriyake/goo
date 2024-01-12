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
	Value int
}

type String struct {
	Value string
}

type Operator struct {
	Value string
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
}

func (p *Parser) parseExpression() (interface{}, error) {
	//fmt.Printf("Token type: %s, Token value: %s\n", p.currentToken.Type, p.currentToken.Literal)

	if p.currentToken.Type == lexer.ILLEGAL {
		return nil, fmt.Errorf("Unexpected token: %s", p.currentToken.Literal)
	}

	switch p.currentToken.Type {
	case lexer.IDENT:
		return Identifier{Value: p.currentToken.Literal}, nil
	case lexer.NUMBER:
		intValue, err := strconv.Atoi(p.currentToken.Literal)
		if err != nil {
			return nil, err
		}
		return Number{Value: intValue}, nil
	case lexer.STRING:
		return String{Value: p.currentToken.Literal}, nil
	case lexer.OPERATOR:
		return Operator{Value: p.currentToken.Literal}, nil
	case lexer.LPAREN:
		return p.parseExpressionList()
	default:
		return nil, fmt.Errorf("Unexpected token: %s", p.currentToken.Literal)
	}
}

func (p *Parser) parseExpressionList() ([]interface{}, error) {
	//fmt.Println("Parsing expression list")

	expressions := []interface{}{}

	p.nextToken()

	for p.currentToken.Type != lexer.RPAREN {
		expression, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expression)

		p.nextToken()
	}

	return expressions, nil
}

func (p *Parser) Parse() (interface{}, error) {
	ast := []interface{}{}

	for p.currentToken.Type != lexer.EOF {
		expression, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		ast = append(ast, expression)
		//fmt.Printf("Parsed expression: %#v\n", expression)

		p.nextToken()
	}

	return ast, nil
}
