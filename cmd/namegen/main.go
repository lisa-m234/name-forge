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
	check := flag.Bool("check", false, "parse the grammar and report errors without generating names")
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
	if *check {
		// Stay quiet on stdout apart from the confirmation so the exit
		// status and one line are all a script or editor hook has to read.
		fmt.Printf("%s: ok\n", path)
		return
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
