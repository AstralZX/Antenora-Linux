package hell

import (
	"fmt"
	"strings"
)

// MaxRecipeLines is the maximum number of lines a Hell recipe may occupy.
// Violating it is blasphemy.
const MaxRecipeLines = 50

// BlasphemyError is returned when a recipe exceeds MaxRecipeLines lines.
type BlasphemyError struct {
	Path string
	Line int
}

func (e BlasphemyError) Error() string {
	return fmt.Sprintf("BLASPHEMY_ERROR: recipe %s exceeds %d lines (%d lines)",
		e.Path, MaxRecipeLines, e.Line)
}

// Package is the root AST node of a Hell recipe.
type Package struct {
	Name        string
	Version     string
	Description string
	Source      string
	Depends     []string
	CFlags      string
	LDFlags     string
	Arches      map[string]*Arch
	Build       []Stmt
	Install     []Stmt
	PostInstall []Stmt
	Binary      *Binary
	Path        string
}

// Arch holds architecture-specific compiler flags and, optionally, build and
// install steps.
type Arch struct {
	CFlags  string
	LDFlags string
	Build   []Stmt
	Install []Stmt
}

// Binary describes the prebuilt binary fallback for a package.
type Binary struct {
	URL    string
	SHA256 string
	Size   string
}

// Stmt is a single instruction or an if/else conditional block.
type Stmt struct {
	Op      string
	Args    []string
	Cond    string
	CondArg string
	Then    []Stmt
	Else    []Stmt
	Line    int
}

// Parse tokenizes and parses Hell source into a Package. It rejects recipes
// longer than MaxRecipeLines with a BlasphemyError.
func Parse(src, path string) (*Package, error) {
	if n := strings.Count(src, "\n") + 1; n > MaxRecipeLines {
		return nil, BlasphemyError{Path: path, Line: n}
	}
	p := &Parser{lex: NewLexer(src), path: path}
	tok, err := p.lex.Next()
	if err != nil {
		return nil, err
	}
	p.cur = tok
	p.peek, err = p.lex.Next()
	if err != nil {
		return nil, err
	}
	return p.parsePackage()
}

// Parser is a recursive-descent parser for Hell.
type Parser struct {
	lex  *Lexer
	cur  Token
	peek Token
	path string
}

func (p *Parser) advance() error {
	p.cur = p.peek
	var err error
	p.peek, err = p.lex.Next()
	return err
}

func (p *Parser) errf(format string, args ...any) error {
	return fmt.Errorf("hell: %s: line %d: %s", p.path, p.cur.Line, fmt.Sprintf(format, args...))
}

func (p *Parser) expectIdent(lit string) error {
	if p.cur.Type != TokenIdent || p.cur.Lit != lit {
		return p.errf("expected %q, got %s %q", lit, p.cur.Type, p.cur.Lit)
	}
	return p.advance()
}

func (p *Parser) expectString() (string, error) {
	if p.cur.Type != TokenString {
		return "", p.errf("expected string, got %s %q", p.cur.Type, p.cur.Lit)
	}
	s := p.cur.Lit
	return s, p.advance()
}

func (p *Parser) expectLBrace() error {
	if p.cur.Type != TokenLBrace {
		return p.errf("expected '{', got %s %q", p.cur.Type, p.cur.Lit)
	}
	return p.advance()
}

func (p *Parser) expectRBrace() error {
	if p.cur.Type != TokenRBrace {
		return p.errf("expected '}', got %s %q", p.cur.Type, p.cur.Lit)
	}
	return p.advance()
}

func (p *Parser) parsePackage() (*Package, error) {
	pkg := &Package{Arches: map[string]*Arch{}, Path: p.path}
	if err := p.expectIdent("package"); err != nil {
		return nil, err
	}
	name, err := p.expectString()
	if err != nil {
		return nil, err
	}
	pkg.Name = name
	if err := p.expectIdent("version"); err != nil {
		return nil, err
	}
	ver, err := p.expectString()
	if err != nil {
		return nil, err
	}
	pkg.Version = ver
	if err := p.expectLBrace(); err != nil {
		return nil, err
	}
	for p.cur.Type != TokenRBrace {
		if p.cur.Type == TokenEOF {
			return nil, p.errf("unexpected end of file, expected '}'")
		}
		if p.cur.Type != TokenIdent {
			return nil, p.errf("unexpected %s %q in package body", p.cur.Type, p.cur.Lit)
		}
		switch p.cur.Lit {
		case "description":
			p.advance()
			if pkg.Description, err = p.expectString(); err != nil {
				return nil, err
			}
		case "source":
			p.advance()
			if pkg.Source, err = p.expectString(); err != nil {
				return nil, err
			}
		case "depends":
			p.advance()
			for p.cur.Type == TokenString {
				pkg.Depends = append(pkg.Depends, p.cur.Lit)
				p.advance()
			}
		case "cflags":
			p.advance()
			if pkg.CFlags, err = p.expectString(); err != nil {
				return nil, err
			}
		case "ldflags":
			p.advance()
			if pkg.LDFlags, err = p.expectString(); err != nil {
				return nil, err
			}
		case "arch":
			name, arch, err := p.parseArch()
			if err != nil {
				return nil, err
			}
			pkg.Arches[name] = arch
		case "build":
			p.advance()
			if pkg.Build, err = p.parseBlock(); err != nil {
				return nil, err
			}
		case "install":
			p.advance()
			if pkg.Install, err = p.parseBlock(); err != nil {
				return nil, err
			}
		case "post_install":
			p.advance()
			if pkg.PostInstall, err = p.parseBlock(); err != nil {
				return nil, err
			}
		case "binary":
			bin, err := p.parseBinary()
			if err != nil {
				return nil, err
			}
			pkg.Binary = bin
		default:
			return nil, p.errf("unexpected keyword %q in package body", p.cur.Lit)
		}
	}
	return pkg, p.expectRBrace()
}

func (p *Parser) parseArch() (string, *Arch, error) {
	if err := p.expectIdent("arch"); err != nil {
		return "", nil, err
	}
	if p.cur.Type != TokenIdent {
		return "", nil, p.errf("expected architecture identifier, got %s %q", p.cur.Type, p.cur.Lit)
	}
	name := p.cur.Lit
	p.advance()
	if err := p.expectLBrace(); err != nil {
		return "", nil, err
	}
	arch := &Arch{}
	for p.cur.Type != TokenRBrace {
		if p.cur.Type != TokenIdent {
			return "", nil, p.errf("unexpected %s %q in arch block", p.cur.Type, p.cur.Lit)
		}
		switch p.cur.Lit {
		case "cflags":
			p.advance()
			var err error
			if arch.CFlags, err = p.expectString(); err != nil {
				return "", nil, err
			}
		case "ldflags":
			p.advance()
			var err error
			if arch.LDFlags, err = p.expectString(); err != nil {
				return "", nil, err
			}
		case "build":
			p.advance()
			var err error
			if arch.Build, err = p.parseBlock(); err != nil {
				return "", nil, err
			}
		case "install":
			p.advance()
			var err error
			if arch.Install, err = p.parseBlock(); err != nil {
				return "", nil, err
			}
		default:
			return "", nil, p.errf("unexpected keyword %q in arch block", p.cur.Lit)
		}
	}
	return name, arch, p.expectRBrace()
}

func (p *Parser) parseBinary() (*Binary, error) {
	if err := p.expectIdent("binary"); err != nil {
		return nil, err
	}
	url, err := p.expectString()
	if err != nil {
		return nil, err
	}
	bin := &Binary{URL: url}
	if err := p.expectLBrace(); err != nil {
		return nil, err
	}
	for p.cur.Type != TokenRBrace {
		if p.cur.Type != TokenIdent {
			return nil, p.errf("unexpected %s %q in binary block", p.cur.Type, p.cur.Lit)
		}
		switch p.cur.Lit {
		case "sha256":
			p.advance()
			if bin.SHA256, err = p.expectString(); err != nil {
				return nil, err
			}
		case "size":
			p.advance()
			if bin.Size, err = p.expectString(); err != nil {
				return nil, err
			}
		default:
			return nil, p.errf("unexpected keyword %q in binary block", p.cur.Lit)
		}
	}
	return bin, p.expectRBrace()
}

func (p *Parser) parseBlock() ([]Stmt, error) {
	if err := p.expectLBrace(); err != nil {
		return nil, err
	}
	stmts, err := p.parseStmts()
	if err != nil {
		return nil, err
	}
	return stmts, p.expectRBrace()
}

func (p *Parser) parseStmts() ([]Stmt, error) {
	var stmts []Stmt
	for p.cur.Type != TokenRBrace {
		if p.cur.Type == TokenEOF {
			return nil, p.errf("unexpected end of file, expected '}'")
		}
		if p.cur.Type != TokenIdent {
			return nil, p.errf("unexpected %s %q in block", p.cur.Type, p.cur.Lit)
		}
		if p.cur.Lit == "if" {
			stmt, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, stmt)
			continue
		}
		stmt, err := p.parseInstruction()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}

var arities = map[string]int{
	"run": 1, "patch": 2, "mkdir": 1, "cp": 2, "rm": 1, "ln": 2, "var": 2,
}

func (p *Parser) parseInstruction() (Stmt, error) {
	op := p.cur.Lit
	line := p.cur.Line
	n, ok := arities[op]
	if !ok {
		return Stmt{}, p.errf("unknown instruction %q", op)
	}
	p.advance()
	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if p.cur.Type != TokenString {
			return Stmt{}, p.errf("%s expects %d string argument(s), got %s %q", op, n, p.cur.Type, p.cur.Lit)
		}
		args = append(args, p.cur.Lit)
		p.advance()
	}
	return Stmt{Op: op, Args: args, Line: line}, nil
}

func (p *Parser) parseIf() (Stmt, error) {
	line := p.cur.Line
	p.advance() // consume "if"
	if p.cur.Type != TokenIdent {
		return Stmt{}, p.errf("expected condition after 'if', got %s %q", p.cur.Type, p.cur.Lit)
	}
	cond := p.cur.Lit
	p.advance()
	var arg string
	if cond == "file_exists" || cond == "env" {
		if p.cur.Type != TokenString {
			return Stmt{}, p.errf("condition %q requires a string argument", cond)
		}
		arg = p.cur.Lit
		p.advance()
	}
	if err := p.expectLBrace(); err != nil {
		return Stmt{}, err
	}
	thenStmts, err := p.parseStmts()
	if err != nil {
		return Stmt{}, err
	}
	if err := p.expectRBrace(); err != nil {
		return Stmt{}, err
	}
	stmt := Stmt{Op: "if", Cond: cond, CondArg: arg, Then: thenStmts, Line: line}
	if p.cur.Type == TokenIdent && p.cur.Lit == "else" {
		p.advance()
		if err := p.expectLBrace(); err != nil {
			return Stmt{}, err
		}
		elseStmts, err := p.parseStmts()
		if err != nil {
			return Stmt{}, err
		}
		if err := p.expectRBrace(); err != nil {
			return Stmt{}, err
		}
		stmt.Else = elseStmts
	}
	return stmt, nil
}
