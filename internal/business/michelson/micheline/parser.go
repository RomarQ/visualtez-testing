package micheline

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/romarq/visualtez-testing/internal/business/michelson/ast"
	"github.com/romarq/visualtez-testing/internal/business/michelson/micheline/token"
)

type Error struct {
	Message string
}

type Parser struct {
	token_position int
	token_kind     token.Kind
	token_text     string
	scanner        Scanner

	trace bool
}

var (
	regex_hex    = regexp.MustCompile("^0x[0-9a-fA-F]+$")
	regex_number = regexp.MustCompile("^-?[0-9]+$")
)

func (p *Parser) Init(scanner Scanner) {
	p.scanner = scanner
}

func (p *Parser) Parse() ast.Node {
	p.next()

	switch kind := p.token_kind; {
	case kind == token.Bytes:
		return p.parseBytes()
	case kind == token.String:
		return p.parseString()
	case kind == token.Int:
		return p.parseInt()
	case kind == token.Identifier:
		return p.parsePrim()
	case kind == token.Open_brace:
		return p.parseSequence()
	default:
		p.scanner.errorf("Unexpected token (%s) as sequence child.", kind.String())
	}

	return nil
}

func (p *Parser) next() {
	p.token_position, p.token_kind, p.token_text = p.scanner.Scan()
	if p.trace {
		fmt.Printf("[Scanner] (%s) with text (%s)\n", p.token_kind.String(), p.token_text)
	}
}

func (p *Parser) parseBytes() ast.Bytes {
	if p.trace {
		fmt.Println("[Parsing|IN] Bytes")
		defer fmt.Println("[Parsing|OUT] Bytes")
	}

	position := p.expect(token.Bytes)
	defer p.next() // Consume next token

	if !isHex(p.token_text) || len(p.token_text)%2 != 0 {
		p.scanner.errorf("Invalid bytes: %s. %v", p.token_text, position)
	}

	return ast.Bytes{
		Position: ast.Position{
			Pos: position,
			End: position + len(p.token_text) - 1,
		},
		Value: p.token_text,
	}
}

func (p *Parser) parseString() ast.String {
	if p.trace {
		fmt.Println("[Parsing|IN] String")
		defer fmt.Println("[Parsing|OUT]  String")
	}

	position := p.expect(token.String)
	defer p.next() // Consume next token

	return ast.String{
		Position: ast.Position{
			Pos: position,
			End: position + len(p.token_text) + /* Count quotes */ 1,
		},
		Value: p.token_text,
	}
}

func (p *Parser) parseInt() ast.Int {
	if p.trace {
		fmt.Println("[Parsing|IN] Int")
		defer fmt.Println("[Parsing|OUT] Int")
	}

	position := p.expect(token.Int)
	defer p.next() // Consume next token

	number, err := strconv.ParseInt(p.token_text, 10, 64)
	if err != nil {
		p.scanner.errorf("Invalid number: %s. %v", p.token_text, position)
	}

	return ast.Int{
		Position: ast.Position{
			Pos: position,
			End: position + len(p.token_text) - 1,
		},
		Value: number,
	}
}

func (p *Parser) parseSequence() ast.Sequence {
	if p.trace {
		fmt.Println("[Parsing|IN] Sequence")
		defer fmt.Println("[Parsing|OUT] Sequence")
	}

	begin := p.expect(token.Open_brace)
	p.next() // Consume next token

	elements := make([]ast.Node, 0)
	for p.token_kind != token.Close_brace {
		switch p.token_kind {
		case token.Bytes:
			elements = append(elements, p.parseBytes())
		case token.String:
			elements = append(elements, p.parseString())
		case token.Int:
			elements = append(elements, p.parseInt())
		case token.Identifier:
			elements = append(elements, p.parsePrim())
		case token.Open_brace:
			elements = append(elements, p.parseSequence())
		default:
			p.scanner.errorf("Unexpected token (%s) as sequence child.", p.token_kind.String())
		}

		if p.token_kind != token.Close_brace {
			p.expect(token.Semi) // Semicolon is used to separate elements in sequences
			p.next()             // Consume next token
		}
	}
	end := p.expect(token.Close_brace)
	defer p.next() // Consume next token

	return ast.Sequence{
		Position: ast.Position{
			Pos: begin,
			End: end,
		},
		Elements: elements,
	}
}

func (p *Parser) parsePrim() ast.Prim {
	if p.trace {
		fmt.Printf("[Parsing|IN] Prim (%s)\n", p.token_text)
		defer fmt.Printf("[Parsing|OUT] Prim (%s)\n", p.token_text)
	}

	begin := p.expect(token.Identifier)
	identifier := p.token_text

	p.next() // Consume next token

	// Check annotations (Annotations can only appear right after an identifier)
	annotations := make([]ast.Annotation, 0)
	for p.token_kind == token.Annot {
		annotations = append(annotations, p.parseAnnotation())
		p.next() // Consume next token
	}

	arguments := make([]ast.Node, 0)

	for {
		switch p.token_kind {
		case token.Bytes:
			arguments = append(arguments, p.parseBytes())
			continue
		case token.String:
			arguments = append(arguments, p.parseString())
			continue
		case token.Int:
			arguments = append(arguments, p.parseInt())
			continue
		case token.Identifier:
			arguments = append(arguments, p.parsePrim())
			continue
		case token.Open_brace:
			arguments = append(arguments, p.parseSequence())
			continue
		}
		break
	}

	return ast.Prim{
		Position: ast.Position{
			Pos: begin,
			End: p.token_position,
		},
		Prim:        identifier,
		Annotations: annotations,
		Arguments:   arguments,
	}
}

func (p *Parser) parseAnnotation() ast.Annotation {
	position := p.expect(token.Annot)

	var annotationKind ast.AnnotationKind
	switch p.token_text[0] {
	case ':':
		annotationKind = ast.TypeAnnotation
	case '@':
		annotationKind = ast.VariableAnnotation
	case '%':
		annotationKind = ast.FieldAnnotation
	default:
		p.scanner.errorf("Unexpected annotation: (%s)", p.token_text)
	}

	return ast.Annotation{
		Position: ast.Position{
			Pos: position,
			End: position + len(p.token_text) - 1,
		},
		Kind:  annotationKind,
		Value: p.token_text,
	}
}

func (p *Parser) expect(kind token.Kind) (pos int) {
	if p.token_kind == kind {
		pos = p.token_position
	} else {
		p.scanner.errorf("Expected token kind (%s), but received (%s).", kind.String(), p.token_kind.String())
	}
	return
}

func isHex(text string) bool    { return regex_hex.MatchString(text) }
func isNumber(text string) bool { return regex_number.MatchString(text) }
