package execsanitize

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
)

var (
	// discardToken is a special replacement string that discards the write operation completely on match
	discardToken = []byte("@discard")
	// discardTokenEscaped is the escaped version of the discardToken)
	discardTokenEscaped = []byte("@@discard")
)

type Sanitizer struct {
	Patterns     []*regexp.Regexp
	Replacements [][]byte
}

func (s *Sanitizer) Sanitize(in []byte) []byte {
	var (
		replacement = []byte{}
		llen        = len(s.Replacements)
	)

	if llen == 1 {
		replacement = s.Replacements[0]
	}

	for i, p := range s.Patterns {
		if llen > 1 {
			replacement = s.Replacements[i]
		}

		if bytes.Equal(replacement, discardToken) && p.Match(in) {
			return []byte{}
		} else if bytes.Equal(replacement, discardTokenEscaped) {
			replacement = discardToken
		}

		in = p.ReplaceAllLiteral(in, replacement)
	}

	return in
}

type SanitizerWriter struct {
	s *Sanitizer
	w io.Writer
}

func (s *Sanitizer) Writer(w io.Writer) io.Writer {
	return &SanitizerWriter{s: s, w: w}
}

func (sw *SanitizerWriter) Write(p []byte) (n int, err error) {
	clean := sw.s.Sanitize(p)
	n = len(p)
	_, err = sw.w.Write(clean)
	return
}

func New(patterns, plainPatterns, replacements []string) (*Sanitizer, error) {
	if len(replacements) > 1 && len(replacements) != (len(patterns)+len(plainPatterns)) {
		return nil, fmt.Errorf("error: mismatched number of replacements")
	}

	var replacementsBytes [][]byte
	if len(replacements) == 1 {
		replacementsBytes = [][]byte{[]byte(replacements[0])}
	} else if len(replacements) > 1 {
		replacementsBytes = make([][]byte, 0, len(replacements))
		for _, r := range replacements {
			replacementsBytes = append(replacementsBytes, []byte(r))
		}
	}

	regexes := make([]*regexp.Regexp, 0, len(patterns)+len(plainPatterns))
	for _, s := range plainPatterns {
		p := regexp.QuoteMeta(s)
		regex, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("error parsing pattern %q: %v\n", p, err)
		}
		regexes = append(regexes, regex)
	}
	for _, p := range patterns {
		regex, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("error parsing pattern %q: %v\n", p, err)
		}
		regexes = append(regexes, regex)
	}

	return &Sanitizer{
		Patterns:     regexes,
		Replacements: replacementsBytes,
	}, nil
}
