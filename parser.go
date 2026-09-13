package namegen

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Position is a 1-indexed line and column in a grammar source file.
type Position struct {
	Line int
	Col  int
}

// ParseError describes exactly where a grammar failed to parse. Error()
// renders the offending source line with a caret under the problem column,
// in the style of a compiler diagnostic, rather than just naming a line
// number and leaving the reader to go find it.
type ParseError struct {
	Pos    Position
	Msg    string
	Source string
}

func (e *ParseError) Error() string {
	lines := strings.Split(e.Source, "\n")
	var snippet string
	if e.Pos.Line >= 1 && e.Pos.Line <= len(lines) {
		snippet = lines[e.Pos.Line-1]
	}
	caretCol := e.Pos.Col - 1
	if caretCol < 0 {
		caretCol = 0
	}
	caret := strings.Repeat(" ", caretCol) + "^"
	return fmt.Sprintf("line %d, col %d: %s\n    %s\n    %s",
		e.Pos.Line, e.Pos.Col, e.Msg, snippet, caret)
}

type termKind int

const (
	termLiteral termKind = iota
	termRef
)

// Term is one piece of an alternative: either literal text to emit or a
// reference to another rule to expand.
type Term struct {
	Kind termKind
	Text string
	Pos  Position
}

// Alternative is one of the possible expansions of a rule. Weight controls
// how often it is chosen relative to its siblings: an alternative written
// with no ":<n>" suffix has a weight of 1, so a plain grammar behaves exactly
// as before weights existed.
type Alternative struct {
	Terms  []Term
	Weight int
}

// Grammar is a parsed, validated set of name rules.
type Grammar struct {
	rules map[string][]Alternative
}

type tokKind int

const (
	tokIdent tokKind = iota
	tokString
	tokEquals
	tokPipe
	tokColon
	tokNumber
)

type token struct {
	kind tokKind
	text string
	col  int
}

// Parse reads a grammar in the namegen format and returns it, or the first
// error found. Every rule reference is checked against the set of defined
// rules, so a typo in a rule name is reported at parse time rather than
// surfacing as an unexplained panic during generation.
func Parse(source string) (*Grammar, error) {
	g := &Grammar{rules: map[string][]Alternative{}}
	defLine := map[string]int{}

	lines := strings.Split(source, "\n")
	currentRule := ""
	haveCurrent := false

	for i, raw := range lines {
		lineNo := i + 1
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		toks, err := tokenizeLine(raw, lineNo, source)
		if err != nil {
			return nil, err
		}
		if len(toks) == 0 {
			continue
		}

		if toks[0].kind == tokPipe {
			if !haveCurrent {
				return nil, &ParseError{
					Pos:    Position{lineNo, toks[0].col},
					Msg:    "'|' continues a rule, but no rule has been started yet",
					Source: source,
				}
			}
			alts, err := splitAndParseAlternatives(toks[1:], lineNo, source)
			if err != nil {
				return nil, err
			}
			g.rules[currentRule] = append(g.rules[currentRule], alts...)
			continue
		}

		if toks[0].kind != tokIdent {
			return nil, &ParseError{
				Pos:    Position{lineNo, toks[0].col},
				Msg:    "expected a rule name here, e.g. 'root = ...'",
				Source: source,
			}
		}
		name := toks[0].text

		if len(toks) < 2 || toks[1].kind != tokEquals {
			col := toks[0].col + len(name)
			if len(toks) >= 2 {
				col = toks[1].col
			}
			return nil, &ParseError{
				Pos:    Position{lineNo, col},
				Msg:    fmt.Sprintf("expected '=' after rule name %q", name),
				Source: source,
			}
		}

		if firstLine, exists := defLine[name]; exists {
			return nil, &ParseError{
				Pos:    Position{lineNo, toks[0].col},
				Msg:    fmt.Sprintf("rule %q is already defined (first defined on line %d)", name, firstLine),
				Source: source,
			}
		}

		alts, err := splitAndParseAlternatives(toks[2:], lineNo, source)
		if err != nil {
			return nil, err
		}

		g.rules[name] = alts
		defLine[name] = lineNo
		currentRule = name
		haveCurrent = true
	}

	if len(g.rules) == 0 {
		return nil, fmt.Errorf("namegen: grammar has no rules")
	}
	if _, ok := g.rules["root"]; !ok {
		return nil, fmt.Errorf(`namegen: grammar has no rule named "root"`)
	}

	for owner, alts := range g.rules {
		for _, alt := range alts {
			for _, t := range alt.Terms {
				if t.Kind != termRef {
					continue
				}
				if _, ok := g.rules[t.Text]; !ok {
					return nil, &ParseError{
						Pos:    t.Pos,
						Msg:    fmt.Sprintf("rule %q references undefined rule %q", owner, t.Text),
						Source: source,
					}
				}
			}
		}
	}

	return g, nil
}

// splitAndParseAlternatives splits a token run on top-level '|' tokens and
// parses each resulting group into an Alternative.
func splitAndParseAlternatives(toks []token, lineNo int, source string) ([]Alternative, error) {
	var groups [][]token
	start := 0
	for i, t := range toks {
		if t.kind == tokPipe {
			groups = append(groups, toks[start:i])
			start = i + 1
		}
	}
	groups = append(groups, toks[start:])

	alts := make([]Alternative, 0, len(groups))
	for _, group := range groups {
		alt, err := parseAlternative(group, lineNo, source)
		if err != nil {
			return nil, err
		}
		alts = append(alts, alt)
	}
	return alts, nil
}

func parseAlternative(toks []token, lineNo int, source string) (Alternative, error) {
	weight := 1
	if n := len(toks); n >= 2 && toks[n-2].kind == tokColon && toks[n-1].kind == tokNumber {
		numTok := toks[n-1]
		val, err := strconv.Atoi(numTok.text)
		if err != nil || val <= 0 {
			return Alternative{}, &ParseError{
				Pos:    Position{lineNo, numTok.col},
				Msg:    fmt.Sprintf("alternative weight must be a positive integer, got %q", numTok.text),
				Source: source,
			}
		}
		weight = val
		toks = toks[:n-2]
	}

	if len(toks) == 0 {
		return Alternative{}, &ParseError{
			Pos:    Position{lineNo, 1},
			Msg:    "empty alternative: expected a literal or a rule name between separators",
			Source: source,
		}
	}
	terms := make([]Term, 0, len(toks))
	for _, t := range toks {
		switch t.kind {
		case tokString:
			terms = append(terms, Term{Kind: termLiteral, Text: t.text, Pos: Position{lineNo, t.col}})
		case tokIdent:
			terms = append(terms, Term{Kind: termRef, Text: t.text, Pos: Position{lineNo, t.col}})
		default:
			return Alternative{}, &ParseError{
				Pos:    Position{lineNo, t.col},
				Msg:    "expected a quoted literal or a rule name here",
				Source: source,
			}
		}
	}
	return Alternative{Terms: terms, Weight: weight}, nil
}

func tokenizeLine(line string, lineNo int, source string) ([]token, error) {
	var toks []token
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		c := runes[i]
		col := i + 1

		switch {
		case c == ' ' || c == '\t':
			i++

		case c == '#':
			i = len(runes)

		case c == '=':
			toks = append(toks, token{tokEquals, "=", col})
			i++

		case c == '|':
			toks = append(toks, token{tokPipe, "|", col})
			i++

		case c == ':':
			toks = append(toks, token{tokColon, ":", col})
			i++

		case unicode.IsDigit(c):
			start := i
			i++
			for i < len(runes) && unicode.IsDigit(runes[i]) {
				i++
			}
			toks = append(toks, token{tokNumber, string(runes[start:i]), start + 1})

		case c == '"':
			start := i
			i++
			var sb strings.Builder
			closed := false
			for i < len(runes) {
				if runes[i] == '\\' && i+1 < len(runes) {
					switch runes[i+1] {
					case '"':
						sb.WriteRune('"')
					case '\\':
						sb.WriteRune('\\')
					case 'n':
						sb.WriteRune('\n')
					case 't':
						sb.WriteRune('\t')
					default:
						return nil, &ParseError{
							Pos:    Position{lineNo, i + 1},
							Msg:    fmt.Sprintf("unknown escape sequence '\\%c'", runes[i+1]),
							Source: source,
						}
					}
					i += 2
					continue
				}
				if runes[i] == '"' {
					closed = true
					i++
					break
				}
				sb.WriteRune(runes[i])
				i++
			}
			if !closed {
				return nil, &ParseError{
					Pos:    Position{lineNo, start + 1},
					Msg:    "unterminated string literal",
					Source: source,
				}
			}
			toks = append(toks, token{tokString, sb.String(), start + 1})

		case isIdentStart(c):
			start := i
			i++
			for i < len(runes) && isIdentPart(runes[i]) {
				i++
			}
			toks = append(toks, token{tokIdent, string(runes[start:i]), start + 1})

		default:
			return nil, &ParseError{
				Pos:    Position{lineNo, col},
				Msg:    fmt.Sprintf("unexpected character %q", c),
				Source: source,
			}
		}
	}

	return toks, nil
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
