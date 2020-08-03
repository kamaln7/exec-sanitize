package main

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizer(t *testing.T) {
	tcs := []struct {
		name         string
		sanitizer    *sanitizer
		in, out      string
		replacements []string
	}{
		{
			in:  "hello delete there",
			out: "hello  there",
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{regexp.MustCompile("delete")},
			},
		},
		{
			in:  "hello there",
			out: "hmm hmm",
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{
					regexp.MustCompile("hello"),
					regexp.MustCompile("there"),
				},
			},
			replacements: []string{"hmm"},
		},
		{
			in:  "hello there",
			out: "ya yeet",
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{
					regexp.MustCompile("hello"),
					regexp.MustCompile("there"),
				},
			},
			replacements: []string{"ya", "yeet"},
		},
		{
			name: "plaintext first regex second",
			in:   "abbc abbbbc abc abbbbbbc",
			out:  "ac a+c a+c a+c",
			sanitizer: must(NewSanitizer(
				[]string{"ab+c"},
				[]string{"abbc"},
				[]string{"ac", "a+c"},
			)),
		},
		{
			in:  "secret",
			out: "",
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{
					regexp.MustCompile("^secret$"),
				},
			},
			replacements: []string{"@discard"},
		},
		{
			in:  "not a secret",
			out: "not a secret",
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{
					regexp.MustCompile("^secret$"),
				},
			},
			replacements: []string{"@discard"},
		},
		{
			in:  "some secret",
			out: "some @discard",
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{
					regexp.MustCompile("secret"),
				},
			},
			replacements: []string{"@@discard"},
		},
	}

	for _, tc := range tcs {
		if tc.replacements != nil {
			tc.sanitizer.replacements = make([][]byte, 0, len(tc.replacements))
			for _, r := range tc.replacements {
				tc.sanitizer.replacements = append(tc.sanitizer.replacements, []byte(r))
			}
		}
	}

	t.Run("sanitize", func(t *testing.T) {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				out := tc.sanitizer.Sanitize([]byte(tc.in))
				assert.Equal(t, tc.out, string(out))
			})
		}
	})

	t.Run("writer", func(t *testing.T) {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				w := tc.sanitizer.Writer(&buf)
				_, err := w.Write([]byte(tc.in))
				require.NoError(t, err)
				assert.Equal(t, tc.out, buf.String())
			})
		}
	})
}

func TestNewSanitizer(t *testing.T) {
	tcs := []struct {
		name             string
		patterns         []string
		plainPatterns    []string
		replacements     []string
		sanitizer        *sanitizer
		wantErr          bool
		fillReplacements bool
	}{
		{
			patterns: []string{"del?ete$"},
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{regexp.MustCompile("del?ete$")},
			},
		},
		{
			patterns:     []string{"del?ete$"},
			replacements: []string{"", ""},
			wantErr:      true,
		},
		{
			patterns:      []string{"del?ete$"},
			plainPatterns: []string{"hmm? ok"},
			replacements:  []string{"", "sure"},
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{
					regexp.MustCompile("hmm\\? ok"),
					regexp.MustCompile("del?ete$"),
				},
			},
			fillReplacements: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sanitizer, err := NewSanitizer(tc.patterns, tc.plainPatterns, tc.replacements)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.fillReplacements {
				tc.sanitizer.replacements = make([][]byte, 0, len(tc.replacements))
				for _, r := range tc.replacements {
					tc.sanitizer.replacements = append(tc.sanitizer.replacements, []byte(r))
				}
			}

			assert.Equal(t, tc.sanitizer, sanitizer)
		})
	}
}

func must(s *sanitizer, err error) *sanitizer {
	if err != nil {
		panic(err)
	}

	return s
}
