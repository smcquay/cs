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
	"sort"
	"sync"
)

// result is a message or error payload
type result struct {
	f   string
	cs  string
	err error
}

// results exists to sort a slice of result
type results []result

func (r results) Len() int           { return len(r) }
func (r results) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r results) Less(i, j int) bool { return r[i].f < r[j].f }

// hashr exists so that we can make a thing that can return valid hash.Hash
// interfaces.
type hashr func() hash.Hash

// hsh figures out which hash algo to use, and distributes the work of hashing
func hsh(files []string, verbose bool) chan result {
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
			r <- result{err: fmt.Errorf("unsupported algorithm: %v (supported: md5, sha1, sha256, sha512)", *algo)}
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
			r <- result{cs: fmt.Sprintf("%x", hsh.Sum(nil)), f: "-"}
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
		res = append(res, compute(h, jobs, verbose))
	}

	o := make(chan result)
	go func() {
		rs := results{}
		for r := range rmerge(res) {
			rs = append(rs, r)
		}
		sort.Sort(rs)
		for _, r := range rs {
			o <- r
		}
		close(o)
	}()
	return o
}

// compute is the checksumming workhorse
func compute(h hashr, jobs chan checksum, verbose bool) chan result {
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
			if verbose {
				fmt.Fprintf(os.Stderr, "%v\n", job.filename)
			}
			r <- result{f: job.filename, cs: fmt.Sprintf("%x", hsh.Sum(nil))}
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
