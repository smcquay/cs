package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"io"
	"os"
)

var algo = flag.String("a", "sha1", "algorithm to use")

func main() {
	flag.Parse()
	files := flag.Args()
	h := sha256.New()
	switch *algo {
	case "sha1", "1":
		h = sha1.New()
	case "sha256", "256":
		h = sha256.New()
	case "sha512", "512":
		h = sha512.New()
	default:
		fmt.Fprintf(os.Stderr, "unsupported algorithm: %v\n", *algo)
		os.Exit(1)
	}

	if len(files) == 0 {
		_, err := io.Copy(h, os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%x  -\n", h.Sum(nil))
	} else {
		for _, name := range files {
			f, err := os.Open(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			h.Reset()
			_, err = io.Copy(h, f)
			f.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			fmt.Printf("%x  %s\n", h.Sum(nil), name)
		}
	}
}
