package namegen

import (
	"errors"
	"strings"
	"testing"
)

func TestTokenizeLine(t *testing.T) {
	toks, err := tokenizeLine(`root = "Ada" | given`, 1, "")
	if err != nil {
		t.Fatalf("tokenizeLine returned error: %v", err)
	}
	want := []token{
		{tokIdent, "root", 1},
		{tokEquals, "=", 6},
		{tokString, "Ada", 8},
		{tokPipe, "|", 14},
		{tokIdent, "given", 16},
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}
	for i, w := range want {
		if toks[i] != w {
			t.Errorf("token %d = %+v, want %+v", i, toks[i], w)
		}
	}
}

func TestTokenizeLineIgnoresTrailingComment(t *testing.T) {
	toks, err := tokenizeLine(`given = "a" # trailing comment`, 1, "")
	if err != nil {
		t.Fatalf("tokenizeLine returned error: %v", err)
	}
	if len(toks) != 3 {
		t.Fatalf("got %d tokens, want 3: %+v", len(toks), toks)
	}
	if toks[2].kind != tokString || toks[2].text != "a" {
		t.Errorf("last token = %+v, want string \"a\"", toks[2])
	}
}

func TestTokenizeLineEscapes(t *testing.T) {
	toks, err := tokenizeLine(`x = "a\nb\tc\"d\\e"`, 1, "")
	if err != nil {
		t.Fatalf("tokenizeLine returned error: %v", err)
	}
	var got string
	for _, tk := range toks {
		if tk.kind == tokString {
			got = tk.text
		}
	}
	want := "a\nb\tc\"d\\e"
	if got != want {
		t.Errorf("decoded string = %q, want %q", got, want)
	}
}

func TestTokenizeLineUnterminatedString(t *testing.T) {
	source := "root = x\ngiven = \"Ada"
	_, err := tokenizeLine(`given = "Ada`, 2, source)
	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("expected *ParseError, got %v", err)
	}
	if perr.Pos.Line != 2 || perr.Pos.Col != 9 {
		t.Errorf("Pos = %+v, want {Line:2 Col:9}", perr.Pos)
	}
}

func TestTokenizeLineUnknownEscape(t *testing.T) {
	_, err := tokenizeLine(`x = "a\qb"`, 1, "")
	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("expected *ParseError, got %v", err)
	}
	if perr.Pos.Col != 7 {
		t.Errorf("Pos.Col = %d, want 7", perr.Pos.Col)
	}
	if !strings.Contains(perr.Msg, "unknown escape sequence") {
		t.Errorf("Msg = %q, want it to mention the unknown escape", perr.Msg)
	}
}

func TestTokenizeLineWeightSuffix(t *testing.T) {
	toks, err := tokenizeLine(`given = "Ada":3 | "Grace"`, 1, "")
	if err != nil {
		t.Fatalf("tokenizeLine returned error: %v", err)
	}
	want := []token{
		{tokIdent, "given", 1},
		{tokEquals, "=", 7},
		{tokString, "Ada", 9},
		{tokColon, ":", 14},
		{tokNumber, "3", 15},
		{tokPipe, "|", 17},
		{tokString, "Grace", 19},
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}
	for i, w := range want {
		if toks[i] != w {
			t.Errorf("token %d = %+v, want %+v", i, toks[i], w)
		}
	}
}

func TestParseValidGrammar(t *testing.T) {
	const src = `
root = given " " family
     | given " the " epithet

given = "Ada" | "Grace"
family = "Lovelace" | "Hopper"
epithet = "Analyst" | "Wise"
`
	g, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(g.rules) != 4 {
		t.Fatalf("got %d rules, want 4: %v", len(g.rules), g.rules)
	}
	rootAlts := g.rules["root"]
	if len(rootAlts) != 2 {
		t.Fatalf("got %d alternatives for root, want 2", len(rootAlts))
	}

	first := rootAlts[0].Terms
	if len(first) != 3 ||
		first[0].Kind != termRef || first[0].Text != "given" ||
		first[1].Kind != termLiteral || first[1].Text != " " ||
		first[2].Kind != termRef || first[2].Text != "family" {
		t.Errorf("root alternative 0 = %+v, not the expected given/\" \"/family sequence", first)
	}

	second := rootAlts[1].Terms
	if len(second) != 3 ||
		second[0].Kind != termRef || second[0].Text != "given" ||
		second[1].Kind != termLiteral || second[1].Text != " the " ||
		second[2].Kind != termRef || second[2].Text != "epithet" {
		t.Errorf("root alternative 1 = %+v, not the expected given/\" the \"/epithet sequence", second)
	}

	if got := len(g.rules["given"]); got != 2 {
		t.Errorf("got %d alternatives for given, want 2", got)
	}
}

func TestParseWeightedAlternatives(t *testing.T) {
	g, err := Parse(`given = "Ada":3 | "Grace" | "Rosalind":2`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	alts := g.rules["given"]
	if len(alts) != 3 {
		t.Fatalf("got %d alternatives, want 3", len(alts))
	}
	wantWeights := []int{3, 1, 2}
	for i, w := range wantWeights {
		if alts[i].Weight != w {
			t.Errorf("alternative %d weight = %d, want %d", i, alts[i].Weight, w)
		}
	}
	if len(alts[0].Terms) != 1 || alts[0].Terms[0].Text != "Ada" {
		t.Errorf("alternative 0 terms = %+v, want just the literal %q", alts[0].Terms, "Ada")
	}
}

func TestParseWeightMustBePositive(t *testing.T) {
	_, err := Parse(`given = "Ada":0`)
	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("Parse returned %v, want *ParseError", err)
	}
	if !strings.Contains(perr.Msg, "positive integer") {
		t.Errorf("Msg = %q, want it to mention a positive integer", perr.Msg)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantMsg string
		line    int
	}{
		{
			name:    "duplicate rule",
			src:     "root = \"a\"\nroot = \"b\"",
			wantMsg: "already defined",
			line:    2,
		},
		{
			name:    "undefined rule reference",
			src:     "root = missing",
			wantMsg: "references undefined rule",
			line:    1,
		},
		{
			name:    "pipe before any rule",
			src:     "| \"a\"",
			wantMsg: "no rule has been started yet",
			line:    1,
		},
		{
			name:    "empty alternative",
			src:     "root = \"a\" |",
			wantMsg: "empty alternative",
			line:    1,
		},
		{
			name:    "missing equals",
			src:     "root \"a\"",
			wantMsg: "expected '='",
			line:    1,
		},
		{
			name:    "expected rule name",
			src:     "\"a\" = \"b\"",
			wantMsg: "expected a rule name",
			line:    1,
		},
		{
			name:    "unexpected character",
			src:     "root = $",
			wantMsg: "unexpected character",
			line:    1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(c.src)
			var perr *ParseError
			if !errors.As(err, &perr) {
				t.Fatalf("Parse(%q) returned %v, want *ParseError", c.src, err)
			}
			if perr.Pos.Line != c.line {
				t.Errorf("Pos.Line = %d, want %d", perr.Pos.Line, c.line)
			}
			if !strings.Contains(perr.Msg, c.wantMsg) {
				t.Errorf("Msg = %q, want it to contain %q", perr.Msg, c.wantMsg)
			}
		})
	}
}

func TestParseEmptyGrammar(t *testing.T) {
	_, err := Parse("")
	if err == nil || !strings.Contains(err.Error(), "no rules") {
		t.Fatalf("Parse(\"\") = %v, want an error about having no rules", err)
	}
}

func TestParseMissingRoot(t *testing.T) {
	_, err := Parse(`foo = "x"`)
	if err == nil || !strings.Contains(err.Error(), `no rule named "root"`) {
		t.Fatalf("Parse without root = %v, want an error about the missing root rule", err)
	}
}

func TestParseErrorMessageFormatsCaret(t *testing.T) {
	src := "root = given \" \" family\ngiven = \"Ada\nfamily = \"Lovelace\" | \"Hopper\""
	_, err := Parse(src)
	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("expected *ParseError, got %v", err)
	}
	got := perr.Error()
	if !strings.Contains(got, "line 2, col 9") {
		t.Errorf("Error() = %q, want it to report line 2, col 9", got)
	}
	if !strings.Contains(got, "^") {
		t.Errorf("Error() = %q, want it to include a caret", got)
	}
}
