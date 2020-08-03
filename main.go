package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
)

func main() {
	log.SetFlags(0)
	ctx, cancel := context.WithCancel(context.Background())

	var (
		patterns      stringSlice
		plainPatterns stringSlice
		replacements  stringSlice
	)
	flag.Var(&patterns, "pattern", "regexp pattern to sanitize. can be set multiple times")
	flag.Var(&plainPatterns, "plain-pattern", "plaintext pattern to sanitize. can be set multiple times")
	flag.Var(&replacements, "replacement", "what to replace matched substrings with. if unset, matches are deleted. if set once, all matches are replaced with the set replacement. if set more than once, there must be a replacement corresponding to each provided pattern (regexp first, then plaintext)")
	flag.Parse()

	if len(os.Args) < 2 {
		log.Printf("usage: exec-sanitize <command> [args...]\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(replacements) > 1 && len(replacements) != (len(patterns)+len(plainPatterns)) {
		log.Printf("error: mismatched number of replacements\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	regexes := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		regex, err := regexp.Compile(p)
		if err != nil {
			log.Printf("error parsing pattern %q: %v\n", p, err)
			os.Exit(1)
		}
		regexes = append(regexes, regex)
	}
	for _, s := range plainPatterns {
		p := regexp.QuoteMeta(s)
		regex, err := regexp.Compile(p)
		if err != nil {
			log.Printf("error parsing pattern %q: %v\n", p, err)
			os.Exit(1)
		}
		regexes = append(regexes, regex)
	}

	replacementsBytes := make([][]byte, len(replacements))
	for i, r := range replacements {
		replacementsBytes[i] = []byte(r)
	}
	s := &sanitizer{
		patterns:     regexes,
		replacements: replacementsBytes,
	}

	args := flag.Args()
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Env = os.Environ()
	c.Stdin = os.Stdin
	c.Stdout = s.Writer(os.Stdout)
	c.Stderr = s.Writer(os.Stderr)

	chanSig := make(chan os.Signal)
	signal.Notify(chanSig, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			select {
			case sig := <-chanSig:
				cancel()
				c.Process.Signal(sig)
			case <-ctx.Done():
				break
			}
		}
	}()

	err := c.Run()
	if err != nil {
		log.Printf("%v\n", err)
		os.Exit(1)
	}
	var (
		exitCode = 0
		exerr    *exec.ExitError
	)
	if errors.As(err, &exerr) {
		exitCode = exerr.ExitCode()
	}

	if err != nil {
		log.Printf("command exited with code %d and error %v\n", exitCode, err)
	}

	cancel()
	os.Exit(exitCode)
}

type stringSlice []string

var _ flag.Value = new(stringSlice)

func (ss *stringSlice) String() string {
	return strings.Join(*ss, ",")
}

func (ss *stringSlice) Set(value string) error {
	(*ss) = append(*ss, value)
	return nil
}

type sanitizer struct {
	patterns     []*regexp.Regexp
	replacements [][]byte
}

func (s *sanitizer) Sanitize(in []byte) []byte {
	var (
		replacement = []byte{}
		llen        = len(s.replacements)
	)

	if llen == 1 {
		replacement = s.replacements[0]
	}

	for i, p := range s.patterns {
		if llen > 1 {
			replacement = s.replacements[i]
		}

		in = p.ReplaceAllLiteral(in, replacement)
	}

	return in
}

type sanitizerWriter struct {
	s *sanitizer
	w io.Writer
}

func (s *sanitizer) Writer(w io.Writer) io.Writer {
	return &sanitizerWriter{s: s, w: w}
}

func (sw *sanitizerWriter) Write(p []byte) (n int, err error) {
	clean := sw.s.Sanitize(p)
	n = len(p)
	_, err = sw.w.Write(clean)
	return
}
