package execsanitize

import (
	"io"
	"regexp"
)

// ReplacerFunc is a function that accept a pattern and its matches, and returns what it should be replaced with
type ReplacerFunc func(string) string

// Sanitizer sanitizes strings according to regex matching rules
type Sanitizer struct {
	Rules map[*regexp.Regexp]ReplacerFunc
}

// Sanitize sanitizes a string using the Sanitizers rules
func (s *Sanitizer) Sanitize(in string) string {
	for pattern, replacer := range s.Rules {
		in = pattern.ReplaceAllStringFunc(in, replacer)
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
