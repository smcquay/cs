package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
)

var algo = flag.String("a", "sha1", "algorithm to use")
var mode = flag.Bool("c", false, "check")
var ngo = flag.Int("n", runtime.NumCPU(), "number of goroutines")

func main() {
	flag.Parse()
	files := flag.Args()
	switch *mode {
	case true:
		c := 0
		for err := range check(files) {
			c++
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		if c > 0 {
			os.Exit(1)
		}
	case false:
		if err := hsh(files); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}

func hsh(files []string) error {
	h := sha256.New()
	switch *algo {
	case "sha1", "1":
		h = sha1.New()
	case "sha256", "256":
		h = sha256.New()
	case "sha512", "512":
		h = sha512.New()
	case "md5":
		h = md5.New()
	default:
		return fmt.Errorf("unsupported algorithm: %v", *algo)
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
	return nil
}
