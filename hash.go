package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"
	"sync"
)

// result is a message or error payload
type result struct {
	msg string
	err error
}

// hashr exists so that we can make a thing that can return valid hash.Hash
// interfaces.
type hashr func() hash.Hash

// hsh figures out which hash algo to use, and distributes the work of hashing
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
			close(r)
		}()
		return r
	}

	if len(files) == 0 {
		r := make(chan result)
		go func() {
			hsh := h()
			_, err := io.Copy(hsh, os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			r <- result{msg: fmt.Sprintf("%x  -", hsh.Sum(nil))}
			close(r)
		}()
		return r
	}

	jobs := make(chan checksum)
	go func() {
		for _, name := range files {
			jobs <- checksum{filename: name}
		}
		close(jobs)
	}()

	res := []<-chan result{}
	for w := 0; w < *ngo; w++ {
		res = append(res, compute(h, jobs))
	}

	return rmerge(res)
}

// compute is the checksumming workhorse
func compute(h hashr, jobs chan checksum) chan result {
	hsh := h()
	r := make(chan result)
	go func() {
		for job := range jobs {
			f, err := os.Open(job.filename)
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
			r <- result{msg: fmt.Sprintf("%x  %s", hsh.Sum(nil), job.filename)}
		}
		close(r)
	}()
	return r
}

// rmerge implements fan-in
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
