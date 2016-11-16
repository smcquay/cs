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

type checksum struct {
	filename string
	hash     hash.Hash
	checksum string
}

func parseCS(line string) (checksum, error) {
	elems := strings.Fields(line)
	if len(elems) != 2 {
		return checksum{}, fmt.Errorf("unexpected content: %d != 2", len(elems))
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
		return checksum{}, fmt.Errorf("unknown format: %q", line)
	}
	return checksum{filename: f, hash: hsh, checksum: cs}, nil
}

type input struct {
	f   io.ReadCloser
	err error
}

type work struct {
	cs  checksum
	err error
}

func streams(files []string) chan input {
	r := make(chan input)

	go func() {
		for _, name := range files {
			f, err := os.Open(name)
			r <- input{f, err}
		}
		if len(files) == 0 {
			r <- input{f: os.Stdin}
		}
		close(r)
	}()

	return r
}

func check(files []string) chan error {
	jobs := make(chan work)

	go func() {
		for stream := range streams(files) {
			if stream.err != nil {
				jobs <- work{err: stream.err}
				break
			}
			s := bufio.NewScanner(stream.f)
			for s.Scan() {
				cs, err := parseCS(s.Text())
				jobs <- work{cs, err}
			}
			stream.f.Close()
			if s.Err() != nil {
				jobs <- work{err: s.Err()}
			}
		}
		close(jobs)
	}()

	results := []<-chan error{}

	workers := 32
	for w := 0; w < workers; w++ {
		results = append(results, compute(jobs))
	}

	return merge(results)
}

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

func compute(jobs chan work) chan error {
	r := make(chan error)
	go func() {
		for job := range jobs {
			if job.err != nil {
				log.Printf("%+v", job.err)
				continue
			}
			f, err := os.Open(job.cs.filename)
			if err != nil {
				r <- fmt.Errorf("open: %v", err)
				continue
			}
			if _, err := io.Copy(job.cs.hash, f); err != nil {
				r <- err
				continue
			}
			f.Close()
			if fmt.Sprintf("%x", job.cs.hash.Sum(nil)) != job.cs.checksum {
				r <- fmt.Errorf("%s: bad", job.cs.filename)
			}
		}
		close(r)
	}()
	return r
}
