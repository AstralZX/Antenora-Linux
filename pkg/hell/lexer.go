// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package hell implements the Hell package recipe language: a tokenizer,
// parser and interpreter for the .hell files consumed by the Dante package
// manager. Hell is a minimalistic cross between Makefile and Lua, designed to
// be impossible to write poorly. A recipe longer than MaxRecipeLines lines is
// rejected with a BlasphemyError.
package hell

import (
	"fmt"
	"strings"
)

// TokenType enumerates the lexical token kinds produced by the Lexer.
type TokenType int

const (
	TokenIdent TokenType = iota
	TokenString
	TokenLBrace
	TokenRBrace
	TokenEOF
)

func (t TokenType) String() string {
	switch t {
	case TokenIdent:
		return "identifier"
	case TokenString:
		return "string"
	case TokenLBrace:
		return "'{'"
	case TokenRBrace:
		return "'}'"
	case TokenEOF:
		return "end of file"
	}
	return "unknown token"
}

// Token is a single lexical unit with its source line for diagnostics.
type Token struct {
	Type TokenType
	Lit  string
	Line int
}

// Lexer tokenizes Hell source. Comments (from '#' to end of line) and all
// whitespace are discarded. Identifiers may contain letters, digits, '_' and
// '-'. Strings are double-quoted and support '\' escapes.
type Lexer struct {
	src  string
	pos  int
	line int
}

// NewLexer returns a Lexer positioned at the start of src.
func NewLexer(src string) *Lexer {
	return &Lexer{src: src, line: 1}
}

// Next returns the next token, skipping whitespace and comments.
func (l *Lexer) Next() (Token, error) {
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == '\n':
			l.line++
			l.pos++
		case c == ' ' || c == '\t' || c == '\r':
			l.pos++
		case c == '#':
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
		case c == '{':
			l.pos++
			return Token{Type: TokenLBrace, Lit: "{", Line: l.line}, nil
		case c == '}':
			l.pos++
			return Token{Type: TokenRBrace, Lit: "}", Line: l.line}, nil
		case c == '"':
			return l.scanString()
		default:
			if isIdentChar(c) {
				start := l.pos
				for l.pos < len(l.src) && isIdentChar(l.src[l.pos]) {
					l.pos++
				}
				return Token{Type: TokenIdent, Lit: l.src[start:l.pos], Line: l.line}, nil
			}
			return Token{}, fmt.Errorf("hell: unexpected character %q at line %d", c, l.line)
		}
	}
	return Token{Type: TokenEOF, Lit: "", Line: l.line}, nil
}

func (l *Lexer) scanString() (Token, error) {
	startLine := l.line
	l.pos++ // consume opening quote
	var sb strings.Builder
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == '"' {
			l.pos++
			return Token{Type: TokenString, Lit: sb.String(), Line: startLine}, nil
		}
		if ch == '\\' && l.pos+1 < len(l.src) {
			l.pos++
			ch = l.src[l.pos]
		}
		sb.WriteByte(ch)
		l.pos++
	}
	return Token{}, fmt.Errorf("hell: unterminated string at line %d", startLine)
}

func isIdentChar(c byte) bool {
	return c == '_' || c == '-' ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}
