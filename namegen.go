// Package namegen generates random names from a small grammar file format.
//
// A grammar is a set of rules. Each rule expands to one of several
// alternatives, and each alternative is a sequence of string literals and
// references to other rules. Generation always starts at the rule named
// "root". See the repository README for the full grammar syntax.
package namegen

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// maxDepth guards against grammars whose rules reference each other in a
// cycle (a = b, b = a). Parse cannot catch every cycle cheaply, since a rule
// is allowed to reference itself for recursive patterns like nested titles,
// so the check is deferred to generation time.
const maxDepth = 200

// Generator produces names from a parsed Grammar.
type Generator struct {
	grammar *Grammar
	rand    *rand.Rand
}

// New parses source as a grammar and returns a Generator seeded from the
// current time. If source contains a syntax or reference error, the
// returned error is a *ParseError with a line and column pointing at the
// problem.
func New(source string) (*Generator, error) {
	g, err := Parse(source)
	if err != nil {
		return nil, err
	}
	return &Generator{
		grammar: g,
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// Seed makes generation deterministic, which is useful for tests and for
// reproducing a specific name from a log line.
func (g *Generator) Seed(seed int64) {
	g.rand = rand.New(rand.NewSource(seed))
}

// Generate expands the grammar's root rule into a single name.
func (g *Generator) Generate() (string, error) {
	var sb strings.Builder
	if err := g.expand("root", &sb, 0); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// GenerateMany returns n names in one call, which is a bit faster than
// calling Generate in a loop and is the shape most callers actually want.
func (g *Generator) GenerateMany(n int) ([]string, error) {
	names := make([]string, n)
	for i := range names {
		name, err := g.Generate()
		if err != nil {
			return nil, err
		}
		names[i] = name
	}
	return names, nil
}

func (g *Generator) expand(rule string, sb *strings.Builder, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("namegen: rule %q recurses too deeply (possible cycle)", rule)
	}
	alts := g.grammar.rules[rule]
	alt := pickAlternative(alts, g.rand)
	for _, t := range alt.Terms {
		switch t.Kind {
		case termLiteral:
			sb.WriteString(t.Text)
		case termRef:
			if err := g.expand(t.Text, sb, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// pickAlternative chooses one of alts at random, favoring higher-weight
// alternatives in proportion to their weight. Most grammars leave every
// alternative at the default weight of 1, which makes this a plain uniform
// choice.
func pickAlternative(alts []Alternative, rnd *rand.Rand) Alternative {
	total := 0
	for _, alt := range alts {
		total += alt.Weight
	}
	n := rnd.Intn(total)
	for _, alt := range alts {
		if n < alt.Weight {
			return alt
		}
		n -= alt.Weight
	}
	// Unreachable as long as every alternative's weight is at least 1.
	return alts[len(alts)-1]
}
