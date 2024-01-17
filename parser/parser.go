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

type TypeAnnotation struct {
	Variable string
	Type     string
}

type FunctionDefinition struct {
	Name       string
	Params     []TypeAnnotation
	ReturnType string
	Body       []interface{}
}

type ReturnStatement struct {
	ReturnValue interface{}
}

type LambdaExpression struct {
	Params []TypeAnnotation
	Body   []interface{}
}

type LambdaCall struct {
	Lambda    interface{}
	Arguments []interface{}
}

type MapExpression struct {
	Lambda    interface{}
	Arguments []interface{}
}

type FilterExpression struct {
	Lambda    interface{}
	Arguments []interface{}
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
		} else if p.currentToken.Literal == "ret" {
			result, err = p.parseReturnStatement()
		} else if p.currentToken.Literal == "let" {
			p.nextToken()
			return p.parseVariableDefinition()
		} else if p.currentToken.Literal == "map" {
			return p.parseMapExpression()
		} else if p.currentToken.Literal == "filter"{
			return p.parseFilterExpression()
		} else if p.peekTokenIs(lexer.LPAREN) {
			return p.parseFunctionCall()
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

		//var operands []interface{}
		firstOperand, err := p.parseOperand()
		if err != nil {
			return nil, err
		}
		//operands = append(operands, firstOperand)
		//fmt.Printf("first operand: %v\n", firstOperand)

		var secondOperand []interface{}
		if !p.peekTokenIs(lexer.RPAREN) && p.peekToken.Type != lexer.EOF {
			p.nextToken()
			operand, err := p.parseOperand()
			if err != nil {
				return nil, err
			}
			//operands = append(operands, secondOperand)
			secondOperand = append(secondOperand, operand)
		}
		//fmt.Printf("second operand: %v\n", secondOperand)

		result = append([]interface{}{operator}, []interface{}{firstOperand}, []interface{}{secondOperand})
	case lexer.LPAREN:
		p.nextToken()
		if p.currentToken.Literal == "map" {
			return p.parseMapExpression()
		} 
		if p.currentToken.Literal == "filter" {
			return p.parseFilterExpression()
		}
		if p.isLambdaExpression() {
			lambdaExpr, err := p.parseLambdaExpression()
			if err != nil {
				return nil, err
			}

			p.nextToken()

			var args []interface{}
			for p.peekTokenIs(lexer.NUMBER) || p.peekTokenIs(lexer.IDENT) {
				p.nextToken()
				arg, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			}

			return LambdaCall{Lambda: lambdaExpr, Arguments: args}, nil
		} else {
			return p.parseParenExpression()
		}
	case lexer.RPAREN:
		p.nextToken()
		return nil, nil
	default:
		err = fmt.Errorf("Unexpected token: %s", p.currentToken.Literal)
	}

	//fmt.Printf("parseExpression - End, Parsed: %+v\n", result)
	return result, err
}

func (p *Parser) isLambdaExpression() bool {
	nextTwoTokens, _ := p.lexer.PeekAhead(2)

	if len(nextTwoTokens) >= 2 {
		return nextTwoTokens[0].Type == lexer.COLON && nextTwoTokens[1].Type == lexer.IDENT
	}

	return false
}

func (p *Parser) parseLambdaExpression() (interface{}, error) {
	if p.currentToken.Type != lexer.LPAREN {
		return nil, fmt.Errorf("expected '(' at the beginning of lambda parameters")
	}

	var params []TypeAnnotation
	p.nextToken()
	for !p.currentTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.EOF) {
		param, err := p.parseLambdaParams()
		if err != nil {
			return nil, err
		}
		params = append(params, param)

		p.nextToken()
	}

	if !p.currentTokenIs(lexer.RPAREN) {
		return nil, fmt.Errorf("expected ')' after lambda parameters")
	}

	if !p.expectPeek(lexer.LAMBDA) {
		return nil, fmt.Errorf("expected '->' after lambda parameters")
	}

	var body []interface{}
	p.nextToken()
	b, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	body = append(body, b)
	p.nextToken()

	return LambdaExpression{Params: params, Body: body}, nil
}

func (p *Parser) parseLambdaParams() (TypeAnnotation, error) {
	if p.currentToken.Type != lexer.IDENT {
		return TypeAnnotation{}, fmt.Errorf("expected variable name, got %s", p.currentToken.Literal)
	}

	varName := p.currentToken.Literal

	p.nextToken()
	if p.currentToken.Type != lexer.COLON {
		return TypeAnnotation{}, fmt.Errorf("expected ':' after variable name, got %s", p.currentToken.Literal)
	}

	p.nextToken()
	if p.currentToken.Type != lexer.IDENT {
		return TypeAnnotation{}, fmt.Errorf("expected variable type identifier after ':', got %s", p.currentToken.Literal)
	}

	varType := p.currentToken.Literal

	return TypeAnnotation{
		Variable: varName,
		Type:     varType,
	}, nil
}

func (p *Parser) parseMapExpression() (interface{}, error) {
	p.nextToken()
	p.nextToken()

	lambdaExpr, err := p.parseLambdaExpression()
	if err != nil {
		return nil, err
	}

	args, err := p.parseExpressionList()
	if err != nil {
		return nil, err
	}

	return MapExpression{
		Lambda:    lambdaExpr,
		Arguments: args,
	}, nil
}

func (p *Parser) parseFilterExpression() (interface{}, error) {
	p.nextToken()
	p.nextToken()

	lambdaExpr, err := p.parseLambdaExpression()
	if err != nil {
		return nil, err
	}

	args, err := p.parseExpressionList()
	if err != nil {
		return nil, err
	}

	return FilterExpression{
		Lambda:    lambdaExpr,
		Arguments: args,
	}, nil
}

func (p *Parser) parseExpressionList() ([]interface{}, error) {
	var expressions []interface{}

	if !p.expectPeek(lexer.LPAREN) {
		return nil, fmt.Errorf("expected '(' to start an expression list, got %s", p.peekToken.Literal)
	}

	p.nextToken()

	for !p.currentTokenIs(lexer.RPAREN) && !p.currentTokenIs(lexer.EOF) {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if expr != nil {
			expressions = append(expressions, expr)
		}

		p.nextToken()
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil, fmt.Errorf("expected ')' at the end of expression list, got %s", p.currentToken.Literal)
	}

	return expressions, nil
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

		if p.peekToken.Type != lexer.RPAREN {
			return nil, fmt.Errorf("expected ')' after nested expression, got %s", p.currentToken.Literal)
		}
		p.nextToken()

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

func (p *Parser) parseLambdaExpression2() (LambdaExpression, error) {
	p.nextToken()

	params, err := p.parseFunctionParameters()
	if err != nil {
		return LambdaExpression{}, err
	}

	p.nextToken()
	var body []interface{}
	b, err := p.parseExpression()
	if err != nil {
		return LambdaExpression{}, err
	}
	body = append(body, b)

	return LambdaExpression{Params: params, Body: body}, nil
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

	var returnType string
	if returnType == "" && len(body) > 0 {
		if lastExpr, ok := body[len(body)-1].(Identifier); ok {
			returnType = lastExpr.Value
		}
	}

	return FunctionDefinition{
		Name:       functionName,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}, nil
}

func (p *Parser) parseVariableDefinition() (TypeAnnotation, error) {
	if p.currentToken.Type != lexer.IDENT {
		return TypeAnnotation{}, fmt.Errorf("expected variable name, got %s", p.currentToken.Literal)
	}

	varName := p.currentToken.Literal

	p.nextToken()
	if p.currentToken.Type != lexer.COLON {
		return TypeAnnotation{}, fmt.Errorf("expected ':' after variable name, got %s", p.currentToken.Literal)
	}

	p.nextToken()
	if p.currentToken.Type != lexer.IDENT {
		return TypeAnnotation{}, fmt.Errorf("expected variable type identifier after ':', got %s", p.currentToken.Literal)
	}

	varType := p.currentToken.Literal

	return TypeAnnotation{
		Variable: varName,
		Type:     varType,
	}, nil
}

func (p *Parser) parseFunctionParameters() ([]TypeAnnotation, error) {
	var params []TypeAnnotation

	p.nextToken()

	for p.currentToken.Type != lexer.RPAREN {
		if p.currentToken.Type == lexer.EOF {
			return nil, fmt.Errorf("unexpected end of file while parsing function parameters")
		}

		if p.currentToken.Type != lexer.IDENT {
			return nil, fmt.Errorf("expected parameter name, got %s", p.currentToken.Literal)
		}

		paramName := p.currentToken.Literal

		var paramType string
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken()
			p.nextToken()
			if p.currentToken.Type != lexer.IDENT {
				return nil, fmt.Errorf("expected parameter type identifier after ':', got %s", p.currentToken.Literal)
			}
			paramType = p.currentToken.Literal
			p.nextToken()
		}

		params = append(params, TypeAnnotation{
			Variable: paramName,
			Type:     paramType,
		})

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

	for !p.currentTokenIs(lexer.RPAREN) && !p.currentTokenIs(lexer.EOF) {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if expr != nil {
			body = append(body, expr)
		}

		if !p.peekTokenIs(lexer.RPAREN) {
			p.nextToken()
		}
	}

	if !p.currentTokenIs(lexer.RPAREN) {
		return nil, fmt.Errorf("expected ')' at the end of function body, got %s", p.currentToken.Literal)
	}

	return body, nil
}

func (p *Parser) currentTokenIs(t string) bool {
	return p.currentToken.Type == t
}

func (p *Parser) parseReturnStatement() (ReturnStatement, error) {
	var ret []interface{}
	p.nextToken()

	returnValue, err := p.parseExpression()
	if err != nil {
		return ReturnStatement{}, err
	}
	if returnValue != nil {
		ret = append(ret, returnValue)
	}

	return ReturnStatement{ReturnValue: ret}, nil
}

func (p *Parser) parseFunctionCall() (interface{}, error) {
	funcName := p.currentToken.Literal
	p.nextToken()

	p.nextToken()
	var args []interface{}
	for !p.currentTokenIs(lexer.RPAREN) && !p.currentTokenIs(lexer.EOF) {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		p.nextToken()
	}

	if !p.currentTokenIs(lexer.RPAREN) {
		return nil, fmt.Errorf("expected ')' at the end of function arguments, got %s", p.currentToken.Literal)
	}

	return []interface{}{Identifier{Value: funcName}, args}, nil
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
