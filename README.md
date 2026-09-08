# name-forge

Most "random name" tools are either a hardcoded list you can't shape, or a
Markov chain you can't control. name-forge is a small grammar file format
plus a Go library and CLI for the middle case: you write down the pieces a
name can be built from and how they combine, and it generates instances of
that pattern.

It also tries hard to be pleasant to author grammars for. Every syntax and
reference error comes back with a line number, a column number, and the
offending source line with a caret pointing at the problem, instead of a
bare "parse error" or a panic three stack frames deep in generation code.

## Grammar format

A grammar is a set of rules. Each rule is a name, an `=`, and one or more
alternatives separated by `|`. An alternative is a sequence of quoted string
literals and bare rule names. Generation starts at the rule named `root`.
Lines starting with `#` are comments, and a line starting with `|` continues
the alternatives of the rule above it.

```
# examples/fantasy.namegen
root = given " " family
     | given " the " epithet

given = "Ael" | "Brend" | "Cael" | "Doran" | "Elowen" | "Fenris" | "Gwyn"
      | "Hollis" | "Isolde" | "Joren" | "Kestrel" | "Liora"

family = "Ashborn" | "Blackwood" | "Cindermoor" | "Duskwalker" | "Emberfall"
       | "Frostvale" | "Grimshaw" | "Hollowmere"

epithet = "Bold" | "Grey" | "Ironhearted" | "Nameless" | "Wanderer" | "Wise"
```

Running this might produce `Kestrel Hollowmere` or `Isolde the Ironhearted`.

## CLI usage

```
go run ./cmd/namegen -n 5 examples/fantasy.namegen
```

```
Kestrel Hollowmere
Ael the Wise
Gwyn Frostvale
Isolde Emberfall
Doran the Bold
```

Pass `-seed` for reproducible output:

```
go run ./cmd/namegen -n 3 -seed 42 examples/fantasy.namegen
```

## Library usage

```go
package main

import (
	"fmt"
	"log"

	namegen "github.com/lisa-m234/name-forge"
)

func main() {
	gen, err := namegen.New(`
		root = "Captain " given
		given = "Ada" | "Grace" | "Rosalind"
	`)
	if err != nil {
		log.Fatal(err)
	}

	name, err := gen.Generate()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name)
}
```

## What a broken grammar looks like

Given this file:

```
root = given " " family
given = "Ada
family = "Lovelace" | "Hopper"
```

name-forge reports:

```
line 2, col 9: unterminated string literal
    given = "Ada
            ^
```

The caret sits directly under the opening quote that never found its match.

## Status

This is an early skeleton: the grammar format, parser, generator, and CLI
all work end to end, but the feature set is deliberately small.

## License

MIT, see [LICENSE](LICENSE).
