package lexer

import (
	_ "fmt"
	"regexp"
	"unicode"
)

type Token struct {
	Type    string
	Literal string
}

const (
	LPAREN     = "LPAREN"
	RPAREN     = "RPAREN"
	COLON = "COLON"
	IDENT      = "IDENT"
	NUMBER     = "NUMBER"
	BOOL       = "BOOL"
	STRING     = "STRING"
	OPERATOR   = "OPERATOR"
	SPACE      = "SPACE"
	WHITESPACE = "WHITESPACE"
	COMMA      = "COMMA"
	COMMENT    = "COMMENT"
	EOF        = "EOF"
	ILLEGAL    = "ILLEGAL"
)

var tokenTypes = []struct {
	token string
	regex string
}{
	{LPAREN, `^\(`},
	{RPAREN, `^\)`},
	{COLON, `^:`},
	{BOOL, `^true|^false`},
	{COMMA, `^,`},
	{NUMBER, `^-?\d+(\.\d+)?`},
	{OPERATOR, `^[-><=+?*]+`},
	{IDENT, `^[a-zA-Z_][a-zA-Z0-9_]*`},
	{STRING, `^'[^']*'`},
	{SPACE, `^\s`},
	{WHITESPACE, `^\s+`},
	{COMMENT, `^;[^\n]*`},
}

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           rune
	currentToken Token
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = rune(l.input[l.readPosition])
	}
	//fmt.Printf("Current char: %c\n", l.ch)
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) NextToken() Token {
	var tok Token

	for unicode.IsSpace(l.ch) {
		l.readChar()
	}

	for _, tt := range tokenTypes {
		regex := regexp.MustCompile(tt.regex)
		if matches := regex.FindString(l.input[l.position:]); matches != "" {
			literal := matches
			tok = Token{Type: tt.token, Literal: literal}
			l.position += len(matches)
			l.readPosition = l.position
			l.readChar()
			return tok
		}
	}

	if l.ch == 0 {
		return Token{Type: EOF, Literal: ""}
	}

	tok = Token{Type: ILLEGAL, Literal: string(l.ch)}
	l.readChar()
	return tok
}
