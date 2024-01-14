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

type FunctionDefinition struct {
    Name   string
    Params []string
    Body   []interface{}
}

type ReturnStatement struct {
    ReturnValue interface{}
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
		if p.currentToken.Literal == "def" {
			return p.parseFunctionDefinition()
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

		var operands []interface{}
		firstOperand, err := p.parseOperand()
		if err != nil {
			return nil, err
		}
		operands = append(operands, firstOperand)

		if !p.peekTokenIs(lexer.RPAREN) && p.peekToken.Type != lexer.EOF {
			p.nextToken()
			secondOperand, err := p.parseOperand()
			if err != nil {
				return nil, err
			}
			operands = append(operands, secondOperand)
		}

		result = append([]interface{}{operator}, operands...)
	case lexer.LPAREN:
		p.nextToken()
		return p.parseParenExpression()
	case lexer.RPAREN:
		p.nextToken()
		return nil, nil
	default:
		err = fmt.Errorf("Unexpected token: %s", p.currentToken.Literal)
	}

	//fmt.Printf("parseExpression - End, Parsed: %+v\n", result)
	return result, err
}

func (p *Parser) parseOperand() (interface{}, error) {
	if p.currentToken.Type == lexer.LPAREN {
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

		if p.currentToken.Type != lexer.RPAREN {
			return nil, fmt.Errorf("expected ')' after nested expression, got %s", p.currentToken.Literal)
		}

		return nestedExpressions, nil
	} else {
		return p.parseExpression()
	}
}

func (p *Parser) parseParenExpression() (interface{}, error) {
	var expressions []interface{}

	for p.currentToken.Type != lexer.RPAREN && p.currentToken.Type != lexer.EOF {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)

		if p.peekToken.Type == lexer.RPAREN {
			break
		}
		p.nextToken()
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil, fmt.Errorf("expected ')' after expression, got %s", p.currentToken.Literal)
	}

	if len(expressions) == 1 {
		return expressions[0], nil
	}
	return expressions, nil
}

func (p *Parser) parseIfStatement() (IfStatement, error) {
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
	ifStmt.ThenBlock = thenBlock

	if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "else" {
		p.nextToken()

		p.nextToken()
		elseBlock, err := p.parseExpression()
		if err != nil {
			return IfStatement{}, err
		}

		if elseExpr, ok := elseBlock.([]interface{}); ok {
			ifStmt.ElseBlock = elseExpr
		} else {
			ifStmt.ElseBlock = []interface{}{elseBlock}
		}
	}

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

func (p *Parser) parseFunctionDefinition() (interface{}, error) {
    p.nextToken()

    if p.currentToken.Type != lexer.IDENT {
        return nil, fmt.Errorf("expected function name, got %s", p.currentToken.Literal)
    }
    functionName := p.currentToken.Literal


    if !p.expectPeek(lexer.LPAREN) {
        return nil, fmt.Errorf("expected '(' before function parameters, got %s", p.peekToken.Literal)
    }

    params, err := p.parseFunctionParameters()
    if err != nil {
        return nil, err
    }

    if p.currentToken.Type != lexer.LPAREN {
        return nil, fmt.Errorf("expected '(' before function body, got %s", p.peekToken.Literal)
    }

	body, err := p.parseFunctionBody()
    if err != nil {
        return FunctionDefinition{}, err
    }

    return FunctionDefinition{
        Name:     functionName,
        Params:   params,
        Body:     body,
    }, nil
}

func (p *Parser) parseFunctionParameters() ([]string, error) {
    var params []string

    p.nextToken()

    for p.currentToken.Type != lexer.RPAREN {
        if p.currentToken.Type == lexer.EOF {
            return nil, fmt.Errorf("unexpected end of file while parsing function parameters")
        }

        if p.currentToken.Type != lexer.IDENT {
            return nil, fmt.Errorf("expected parameter name, got %s", p.currentToken.Literal)
        }

        params = append(params, p.currentToken.Literal)

        p.nextToken()

        if p.currentToken.Type == lexer.COMMA {
            p.nextToken()
        }
    }

    if p.currentToken.Type != lexer.RPAREN {
        return nil, fmt.Errorf("expected ')' after function parameters, got %s", p.currentToken.Literal)
    }
    p.nextToken() 

    return params, nil
}

func (p *Parser) parseFunctionBody() ([]interface{}, error) {
    var body []interface{}

    p.nextToken()

    for p.currentToken.Type != lexer.RPAREN && p.currentToken.Type != lexer.EOF {
        expr, err := p.parseParenExpression()
        if err != nil {
            return nil, err
        }
        body = append(body, expr)

        p.nextToken()
    }

    if !p.expectPeek(lexer.RPAREN) {
        return nil, fmt.Errorf("expected ')' at the end of function body, got %s", p.currentToken.Literal)
    }

    return body, nil
}

func (p *Parser) parseReturnStatement() (ReturnStatement, error) {
    p.nextToken()

    returnValue, err := p.parseExpression()
    if err != nil {
        return ReturnStatement{}, err
    }

    return ReturnStatement{ReturnValue: returnValue}, nil
}

func (p *Parser) Parse() (interface{}, error) {
	//fmt.Println("Parse - Start")
	var ast []interface{}

	for p.currentToken.Type != lexer.EOF {
		expression, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if expression != nil {
			ast = append(ast, expression)
		}
		//fmt.Printf("Parsed expression: %v\n", expression)
		p.nextToken()
	}

	//fmt.Println("Parse - End")
	return ast, nil
}
