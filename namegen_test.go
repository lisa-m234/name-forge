package namegen

import (
	"errors"
	"strings"
	"testing"
)

const testGrammar = `
root = given " " family
     | given " the " epithet

given = "Ada" | "Grace" | "Rosalind"
family = "Lovelace" | "Hopper" | "Franklin"
epithet = "Bold" | "Wise"
`

var (
	testGivens   = []string{"Ada", "Grace", "Rosalind"}
	testFamilies = []string{"Lovelace", "Hopper", "Franklin"}
	testEpithets = []string{"Bold", "Wise"}
)

// isValidTestName reports whether name matches one of the two shapes
// testGrammar's root rule can produce: "<given> <family>" or
// "<given> the <epithet>".
func isValidTestName(name string) bool {
	for _, g := range testGivens {
		rest, ok := strings.CutPrefix(name, g)
		if !ok {
			continue
		}
		if epithetPart, ok := strings.CutPrefix(rest, " the "); ok {
			for _, e := range testEpithets {
				if epithetPart == e {
					return true
				}
			}
		}
		if familyPart, ok := strings.CutPrefix(rest, " "); ok {
			for _, f := range testFamilies {
				if familyPart == f {
					return true
				}
			}
		}
	}
	return false
}

func TestGenerateProducesValidNames(t *testing.T) {
	gen, err := New(testGrammar)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	gen.Seed(1)

	for i := 0; i < 200; i++ {
		name, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}
		if !isValidTestName(name) {
			t.Fatalf("Generate produced %q, which matches neither grammar shape", name)
		}
	}
}

func TestGenerateIsDeterministicForASeed(t *testing.T) {
	gen1, err := New(testGrammar)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	gen1.Seed(42)

	gen2, err := New(testGrammar)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	gen2.Seed(42)

	names1, err := gen1.GenerateMany(50)
	if err != nil {
		t.Fatalf("GenerateMany returned error: %v", err)
	}
	names2, err := gen2.GenerateMany(50)
	if err != nil {
		t.Fatalf("GenerateMany returned error: %v", err)
	}

	for i := range names1 {
		if names1[i] != names2[i] {
			t.Fatalf("name %d differs between equally-seeded generators: %q vs %q", i, names1[i], names2[i])
		}
	}
}

func TestGenerateMany(t *testing.T) {
	gen, err := New(testGrammar)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	gen.Seed(7)

	names, err := gen.GenerateMany(10)
	if err != nil {
		t.Fatalf("GenerateMany returned error: %v", err)
	}
	if len(names) != 10 {
		t.Fatalf("got %d names, want 10", len(names))
	}
	for _, name := range names {
		if !isValidTestName(name) {
			t.Errorf("GenerateMany produced %q, which matches neither grammar shape", name)
		}
	}
}

func TestGenerateManyZero(t *testing.T) {
	gen, err := New(testGrammar)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	names, err := gen.GenerateMany(0)
	if err != nil {
		t.Fatalf("GenerateMany returned error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("got %d names, want 0", len(names))
	}
}

func TestGenerateDetectsUnboundedRecursion(t *testing.T) {
	gen, err := New(`root = "a" root`)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	gen.Seed(1)

	_, err = gen.Generate()
	if err == nil {
		t.Fatal("Generate returned no error for a rule that always recurses")
	}
	if !strings.Contains(err.Error(), "recurses too deeply") {
		t.Errorf("error = %v, want it to mention excessive recursion", err)
	}
}

func TestNewReturnsParseError(t *testing.T) {
	_, err := New(`root = missing`)
	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("New returned %v, want a *ParseError", err)
	}
	if perr.Pos.Line != 1 {
		t.Errorf("Pos.Line = %d, want 1", perr.Pos.Line)
	}
}
