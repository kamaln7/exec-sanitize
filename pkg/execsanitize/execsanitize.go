package execsanitize

import (
	"io"
	"regexp"
)

const (
	// DiscardToken is a special replacement string that discards the write operation completely on match
	DiscardToken = "\x03\x03"
)

// ReplacerFunc is a function that accept a match and returns its replacement
type ReplacerFunc func(string) string

// Sanitizer sanitizes strings according to regex matching rules
type Sanitizer struct {
	Rules []*Rule
}

type Rule struct {
	Pattern  *regexp.Regexp
	Replacer ReplacerFunc
}

// Sanitize sanitizes a string using the Sanitizers rules
func (s *Sanitizer) Sanitize(in string) string {
	var discard bool
	for _, rule := range s.Rules {
		in = rule.Pattern.ReplaceAllStringFunc(in, func(in string) (out string) {
			s := rule.Replacer(in)
			if s == DiscardToken {
				discard = true
			}

			return s
		})

		if discard {
			return ""
		}
	}

	return in
}

// SanitizerWriter is a wrapping writer that sanitizes all input
type SanitizerWriter struct {
	s *Sanitizer
	w io.Writer
}

// Writer wraps a writer with a sanitizer
func (s *Sanitizer) Writer(w io.Writer) io.Writer {
	return &SanitizerWriter{s: s, w: w}
}

// Write sanitizes bytes and passes them through to the underlying writer
func (sw *SanitizerWriter) Write(p []byte) (n int, err error) {
	clean := sw.s.Sanitize(string(p))
	n = len(p)
	_, err = sw.w.Write([]byte(clean))
	return
}
