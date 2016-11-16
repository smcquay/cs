package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// input contains a file-ish piece of work to perform
type input struct {
	f   io.ReadCloser
	err error
}

// checksum contains the path to a file, a way to hash it, and the results of
// the hash
type checksum struct {
	filename string
	hash     hash.Hash
	checksum string
	err      error
}

// check is the entry point for -c operation.
func check(args []string) chan error {
	jobs := make(chan checksum)

	go func() {
		for i := range toInput(args) {
			if i.err != nil {
				jobs <- checksum{err: i.err}
				break
			}
			s := bufio.NewScanner(i.f)
			for s.Scan() {
				jobs <- parseCS(s.Text())
			}
			i.f.Close()
			if s.Err() != nil {
				jobs <- checksum{err: s.Err()}
			}
		}
		close(jobs)
	}()

	results := []<-chan error{}

	for w := 0; w < *ngo; w++ {
		results = append(results, verify(jobs))
	}

	return merge(results)
}

// toInput converts args to a stream of input
func toInput(args []string) chan input {
	r := make(chan input)

	go func() {
		for _, name := range args {
			f, err := os.Open(name)
			r <- input{f, err}
		}
		if len(args) == 0 {
			r <- input{f: os.Stdin}
		}
		close(r)
	}()

	return r
}

// parseCS picks apart a line from a checksum file and returns everything
// needed to perform a checksum.
func parseCS(line string) checksum {
	elems := strings.Fields(line)
	if len(elems) != 2 {
		return checksum{err: fmt.Errorf("unexpected content: %d != 2", len(elems))}
	}
	cs, f := elems[0], elems[1]
	var hsh hash.Hash
	switch len(cs) {
	case 32:
		hsh = md5.New()
	case 40:
		hsh = sha1.New()
	case 64:
		hsh = sha256.New()
	case 128:
		hsh = sha512.New()
	default:
		return checksum{err: fmt.Errorf("unknown format: %q", line)}
	}
	return checksum{filename: f, hash: hsh, checksum: cs}
}

// verify does grunt work of verifying a stream of jobs (filenames).
func verify(jobs chan checksum) chan error {
	r := make(chan error)
	go func() {
		for job := range jobs {
			if job.err != nil {
				log.Printf("%+v", job.err)
				continue
			}
			f, err := os.Open(job.filename)
			if err != nil {
				r <- err
				continue
			}
			if _, err := io.Copy(job.hash, f); err != nil {
				r <- err
				continue
			}
			f.Close()
			if fmt.Sprintf("%x", job.hash.Sum(nil)) != job.checksum {
				r <- fmt.Errorf("%s: bad", job.filename)
			}
		}
		close(r)
	}()
	return r
}

// merge is simple error fan-in
func merge(cs []<-chan error) chan error {
	out := make(chan error)

	var wg sync.WaitGroup

	output := func(c <-chan error) {
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
