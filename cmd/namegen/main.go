// Command namegen generates random names from a grammar file.
package main

import (
	"flag"
	"fmt"
	"os"

	namegen "github.com/lisa-m234/name-forge"
)

func main() {
	n := flag.Int("n", 1, "number of names to generate")
	seed := flag.Int64("seed", 0, "random seed; 0 picks a seed from the current time")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: namegen [flags] <grammar-file>\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}
	path := args[0]

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "namegen: %v\n", err)
		os.Exit(1)
	}

	gen, err := namegen.New(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		os.Exit(1)
	}
	if *seed != 0 {
		gen.Seed(*seed)
	}

	for i := 0; i < *n; i++ {
		name, err := gen.Generate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "namegen: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(name)
	}
}
