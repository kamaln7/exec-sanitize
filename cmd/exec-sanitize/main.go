package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kamaln7/exec-sanitize/pkg/execsanitize"
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
	flag.Var(&replacements, "replacement", "what to replace matched substrings with. if unset, matches are deleted. if set once, all matches are replaced with the set replacement. if set more than once, there must be a replacement corresponding to each provided pattern (plain patterns first, then regex patterns)")
	flag.Parse()

	if len(os.Args) < 2 {
		log.Printf("usage: exec-sanitize <command> [args...]\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	s, err := execsanitize.New(patterns, plainPatterns, replacements)
	if err != nil {
		log.Printf("%v\n", err)
		os.Exit(1)
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

	err = c.Run()
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
