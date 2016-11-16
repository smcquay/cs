package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"runtime"
	"sync"
)

var algo = flag.String("a", "sha1", "algorithm to use")
var mode = flag.Bool("c", false, "check")
var ngo = flag.Int("n", runtime.NumCPU(), "number of goroutines")

func main() {
	flag.Parse()
	files := flag.Args()
	switch *mode {
	case true:
		ec := 0
		for err := range check(files) {
			ec++
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		if ec > 0 {
			os.Exit(1)
		}
	case false:
		ec := 0
		for res := range hsh(files) {
			if res.err != nil {
				ec++
				fmt.Fprintf(os.Stderr, "%v\n", res.err)
			} else {
				fmt.Printf("%v\n", res.msg)
			}
		}
		if ec > 0 {
			os.Exit(1)
		}
	}
}

type hashr func() hash.Hash

func hsh(files []string) chan result {
	var h hashr
	switch *algo {
	case "sha1", "1":
		h = sha1.New
	case "sha256", "256":
		h = sha256.New
	case "sha512", "512":
		h = sha512.New
	case "md5":
		h = md5.New
	default:
		r := make(chan result)
		go func() {
			r <- result{err: fmt.Errorf("unsupported algorithm: %v", *algo)}
		}()
		return r
	}

	if len(files) == 0 {
		hsh := h()
		_, err := io.Copy(hsh, os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%x  -\n", hsh.Sum(nil))
		return nil
	}

	jobs := make(chan work)
	go func() {
		for _, name := range files {
			jobs <- work{cs: checksum{filename: name}}
		}
		close(jobs)
	}()

	res := []<-chan result{}
	for w := 0; w < *ngo; w++ {
		res = append(res, compute(h, jobs))
	}

	return rmerge(res)
}

type result struct {
	msg string
	err error
}

func compute(h hashr, jobs chan work) chan result {
	hsh := h()
	r := make(chan result)
	go func() {
		for job := range jobs {
			f, err := os.Open(job.cs.filename)
			if err != nil {
				r <- result{err: err}
				continue
			}
			hsh.Reset()
			_, err = io.Copy(hsh, f)
			f.Close()
			if err != nil {
				r <- result{err: err}
				continue
			}
			r <- result{msg: fmt.Sprintf("%x  %s", hsh.Sum(nil), job.cs.filename)}
		}
		close(r)
	}()
	return r
}

func rmerge(cs []<-chan result) chan result {
	out := make(chan result)

	var wg sync.WaitGroup

	output := func(c <-chan result) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
