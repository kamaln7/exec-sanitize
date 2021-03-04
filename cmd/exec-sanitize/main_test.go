package main

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
