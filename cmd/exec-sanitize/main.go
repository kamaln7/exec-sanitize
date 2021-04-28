package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/kamaln7/exec-sanitize/v2/pkg/execsanitize"
)

var errPrintUsage = fmt.Errorf("u")

const usageText = `usage: exec-sanitize <patterns and replacements> -- <command> [args...]

each pattern must be directly followed with replacement. a replacement value of "@discard" deletes the line entirely.

	-log value
		optional directory to log substituted strings as numbered files. if set, replacements will have the first asterisk * replaced with the log item number
	-p:regex value
		regexp pattern to sanitize.
	-p:plain value
		plaintext pattern to sanitize.
	-r value
		what to replace matched substrings with.
`

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr, os.Args))
}

func run(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(args) < 2 {
		fmt.Fprint(stderr, usageText)
		return 1
	}

	parsedArgs, err := parseArgs(args[1:])
	if err != nil {
		if err == errPrintUsage {
			fmt.Fprint(stderr, usageText)
			return 0
		}

		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	rules, err := parsedArgs.Rules()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	s := &execsanitize.Sanitizer{Rules: rules}

	c := exec.CommandContext(ctx, parsedArgs.cmd, parsedArgs.cmdArgs...)
	c.Env = os.Environ()
	c.Stdin = stdin
	c.Stdout = s.Writer(stdout)
	c.Stderr = s.Writer(stderr)

	chanSig := make(chan os.Signal)
	signal.Notify(chanSig, os.Interrupt, syscall.SIGTERM)
	go func() {
	loop:
		for {
			select {
			case sig := <-chanSig:
				_ = c.Process.Signal(sig)
				cancel()
			case <-ctx.Done():
				break loop
			}
		}
	}()

	err = c.Run()
	if err != nil {
		var (
			exitCode = 1
			exerr    *exec.ExitError
		)
		if errors.As(err, &exerr) {
			exitCode = exerr.ExitCode()
		} else {
			fmt.Fprintf(stderr, "\ncommand exited with error %v\n", err)
			return exitCode
		}

		fmt.Fprintf(stderr, "\ncommand exited with code %d\n", exitCode)
		return exitCode
	}

	return 0
}

// this is an intermediate step before the replacements are turned into ReplacerFuncs
// to make things easier to test
type parsedArgs struct {
	rules   []parsedRule
	cmd     string
	cmdArgs []string
	logPath string
}

type parsedRule struct {
	pattern, replacement string
}

func parseArgs(args []string) (*parsedArgs, error) {
	parsed := &parsedArgs{}

	args = append([]string{}, args...)
	var rule string
	for len(args) > 0 {
		arg := args[0]
		args = args[1:]

		// end of args
		if arg == "--" {
			break
		}

		// validation rules
		if arg == "-r" || strings.HasPrefix(arg, "-r:") && rule == "" {
			return nil, fmt.Errorf("replacement must be directly preceeded by a pattern")
		}

		if strings.HasPrefix(arg, "-p:") && rule != "" {
			return nil, fmt.Errorf("pattern must be followed with a replacement")
		}

		// args that don't take values
		switch arg {
		case "--help":
			return nil, errPrintUsage
		case "-r:discard":
			parsed.rules = append(parsed.rules, parsedRule{pattern: rule, replacement: execsanitize.DiscardToken})
			rule = ""
			continue
		}

		// args that take values
		if len(args) == 0 {
			return nil, fmt.Errorf("unbalanced number of args")
		}

		value := args[0]
		args = args[1:]
		switch arg {
		case "-log":
			parsed.logPath = value
		case "-p:regex":
			rule = value
		case "-p:plain":
			rule = regexp.QuoteMeta(value)
		case "-r":
			parsed.rules = append(parsed.rules, parsedRule{pattern: rule, replacement: value})
			rule = ""
		default:
			return nil, fmt.Errorf("unrecognized flag %s", arg)
		}
	}

	if len(args) > 0 {
		parsed.cmd = args[0]
		parsed.cmdArgs = args[1:]
	}

	return parsed, nil
}

func (a *parsedArgs) Rules() ([]*execsanitize.Rule, error) {
	rules := make([]*execsanitize.Rule, 0, len(a.rules))

	var loggerIdx int
	withLogger := func(r execsanitize.ReplacerFunc) execsanitize.ReplacerFunc {
		if a.logPath == "" {
			return r
		}

		return func(in string) string {
			s := r(in)

			idx := loggerIdx
			loggerIdx++

			_ = ioutil.WriteFile(filepath.Join(a.logPath, fmt.Sprint(idx)), []byte(in), 0644)

			s = strings.Replace(s, "*", fmt.Sprint(idx), 1)
			return s
		}
	}

	for _, rule := range a.rules {
		rule := rule

		rgxp, err := regexp.Compile(rule.pattern)
		if err != nil {
			return nil, fmt.Errorf("parsing pattern %s: %w", rule.pattern, err)
		}

		rules = append(rules, &execsanitize.Rule{
			Pattern: rgxp,
			Replacer: withLogger(func(in string) string {
				return rule.replacement
			}),
		})
	}

	return rules, nil
}
