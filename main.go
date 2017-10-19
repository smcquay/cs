package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

var algo = flag.String("a", "sha256", "algorithm to use")
var mode = flag.Bool("c", false, "check")
var ngo = flag.Int("n", runtime.NumCPU(), "number of goroutines")
var verbose = flag.Bool("v", false, "vebose")

func main() {
	flag.Parse()
	files := flag.Args()
	switch *mode {
	case true:
		ec := 0
		for err := range check(files, *verbose) {
			ec++
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		if ec > 0 {
			os.Exit(1)
		}
	case false:
		ec := 0
		for res := range hsh(files, *verbose) {
			if res.err != nil {
				ec++
				fmt.Fprintf(os.Stderr, "%v\n", res.err)
			} else {
				fmt.Printf("%v   %v\n", res.cs, res.f)
			}
		}
		if ec > 0 {
			os.Exit(1)
		}
	}
}
