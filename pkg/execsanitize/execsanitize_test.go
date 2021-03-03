package execsanitize

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
		sanitizer    *Sanitizer
		in, out      string
		replacements []string
	}{
		{
			in:  "hello delete there",
			out: "hello  there",
			sanitizer: &Sanitizer{
				Patterns: []*regexp.Regexp{regexp.MustCompile("delete")},
			},
		},
		{
			in:  "hello there",
			out: "hmm hmm",
			sanitizer: &Sanitizer{
				Patterns: []*regexp.Regexp{
					regexp.MustCompile("hello"),
					regexp.MustCompile("there"),
				},
			},
			replacements: []string{"hmm"},
		},
		{
			in:  "hello there",
			out: "ya yeet",
			sanitizer: &Sanitizer{
				Patterns: []*regexp.Regexp{
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
			sanitizer: must(New(
				[]string{"ab+c"},
				[]string{"abbc"},
				[]string{"ac", "a+c"},
			)),
		},
		{
			in:  "secret",
			out: "",
			sanitizer: &Sanitizer{
				Patterns: []*regexp.Regexp{
					regexp.MustCompile("^secret$"),
				},
			},
			replacements: []string{"@discard"},
		},
		{
			in:  "not a secret",
			out: "not a secret",
			sanitizer: &Sanitizer{
				Patterns: []*regexp.Regexp{
					regexp.MustCompile("^secret$"),
				},
			},
			replacements: []string{"@discard"},
		},
		{
			in:  "some secret",
			out: "some @discard",
			sanitizer: &Sanitizer{
				Patterns: []*regexp.Regexp{
					regexp.MustCompile("secret"),
				},
			},
			replacements: []string{"@@discard"},
		},
	}

	for _, tc := range tcs {
		if tc.replacements != nil {
			tc.sanitizer.Replacements = make([][]byte, 0, len(tc.replacements))
			for _, r := range tc.replacements {
				tc.sanitizer.Replacements = append(tc.sanitizer.Replacements, []byte(r))
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
		sanitizer        *Sanitizer
		wantErr          bool
		fillReplacements bool
	}{
		{
			patterns: []string{"del?ete$"},
			sanitizer: &Sanitizer{
				Patterns: []*regexp.Regexp{regexp.MustCompile("del?ete$")},
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
			sanitizer: &Sanitizer{
				Patterns: []*regexp.Regexp{
					regexp.MustCompile("hmm\\? ok"),
					regexp.MustCompile("del?ete$"),
				},
			},
			fillReplacements: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sanitizer, err := New(tc.patterns, tc.plainPatterns, tc.replacements)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.fillReplacements {
				tc.sanitizer.Replacements = make([][]byte, 0, len(tc.replacements))
				for _, r := range tc.replacements {
					tc.sanitizer.Replacements = append(tc.sanitizer.Replacements, []byte(r))
				}
			}

			assert.Equal(t, tc.sanitizer, sanitizer)
		})
	}
}

func must(s *Sanitizer, err error) *Sanitizer {
	if err != nil {
		panic(err)
	}

	return s
}
