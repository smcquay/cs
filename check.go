package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"strings"
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

func check(files []string) error {
	streams := []io.ReadCloser{}
	defer func() {
		for _, stream := range streams {
			stream.Close()
		}
	}()

	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		streams = append(streams, f)
	}
	if len(files) == 0 {
		streams = append(streams, os.Stdin)
	}

	jobs := []checksum{}
	for _, stream := range streams {
		s := bufio.NewScanner(stream)
		for s.Scan() {
			cs, err := parseCS(s.Text())
			if err != nil {
				return err
			}
			jobs = append(jobs, cs)
		}
		if s.Err() != nil {
			return s.Err()
		}
	}

	errs := 0
	for _, job := range jobs {
		f, err := os.Open(job.filename)
		if err != nil {
			return fmt.Errorf("open: %v", err)
		}
		if _, err := io.Copy(job.hash, f); err != nil {
			log.Printf("%+v", err)
		}
		f.Close()
		if fmt.Sprintf("%x", job.hash.Sum(nil)) == job.checksum {
			fmt.Printf("%s: OK\n", job.filename)
		} else {
			errs++
			fmt.Fprintf(os.Stderr, "%s: bad\n", job.filename)
		}
	}

	var err error
	if errs != 0 {
		err = errors.New("bad files found")
	}
	return err
}
