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
		name      string
		sanitizer *sanitizer
		in, out   string
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
				replacements: [][]byte{[]byte("hmm")},
			},
		},
		{
			in:  "hello there",
			out: "ya yeet",
			sanitizer: &sanitizer{
				patterns: []*regexp.Regexp{
					regexp.MustCompile("hello"),
					regexp.MustCompile("there"),
				},
				replacements: [][]byte{
					[]byte("ya"),
					[]byte("yeet"),
				},
			},
		},
	}

	t.Run("sanitize", func(t *testing.T) {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				out := tc.sanitizer.Sanitize([]byte(tc.in))
				assert.Equal(t, []byte(tc.out), out)
			})
		}
	})

	t.Run("writer", func(t *testing.T) {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				var w bytes.Buffer
				sw := tc.sanitizer.Writer(&w)
				_, err := sw.Write([]byte(tc.in))
				require.NoError(t, err)
				assert.Equal(t, []byte(tc.out), w.Bytes())
			})
		}
	})
}
