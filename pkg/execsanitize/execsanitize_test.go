package execsanitize

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizer(t *testing.T) {
	tcs := []struct {
		name      string
		sanitizer *Sanitizer
		tests     [][]string
	}{
		{
			name: "simple",
			tests: [][]string{
				{"hello delete there", "hello - there"},
				{"hello delete delete there", "hello - - there"},
				{"whole line", "this line was -d"},
			},
			sanitizer: &Sanitizer{
				Rules: makeRules(
					regexp.MustCompile(`^whole line$`), "this line was deleted",
					"delete", "-",
				),
			},
		},
		{
			name: "simple reversed",
			tests: [][]string{
				{"hello delete there", "hello - there"},
				{"hello delete delete there", "hello - - there"},
				{"whole line", "this line was deleted"},
			},
			sanitizer: &Sanitizer{
				Rules: makeRules(
					"delete", "-",
					regexp.MustCompile(`^whole line$`), "this line was deleted",
				),
			},
		},
		{
			name: "capture groups",
			tests: [][]string{
				{"a b c hello! d e hey f.", "a b c In this house we say 'ello and not hello! d e In this house we say 'ello and not hey f."},
			},
			sanitizer: &Sanitizer{
				Rules: makeRules(
					regexp.MustCompile(`he(y|llo)`), func(s string) string {
						return fmt.Sprintf("In this house we say 'ello and not %s", s)
					},
				),
			},
		},
		{
			name: "vowel counter",
			tests: [][]string{
				{"hello there", "h<1:e>ll<2:o> th<3:e>r<4:e>"},
				{"the counter should keep going", "th<5:e> c<6:o><7:u>nt<8:e>r sh<9:o><10:u>ld k<11:e><12:e>p g<13:o><14:i>ng"},
			},
			sanitizer: &Sanitizer{
				Rules: makeRules(
					regexp.MustCompile(`[aieou]`), counterReplacer(),
				),
			},
		},
		{
			name: "discard",
			tests: [][]string{
				{"hello secret there", ""},
				{"secret", ""},
				{"hi it's a secret message", ""},
				{"hi it's a public message", "hello it's a public message"},
			},
			sanitizer: &Sanitizer{
				Rules: makeRules(
					"hi", "hello",
					"secret", DiscardToken,
				),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.tests {
				in, want := c[0], c[1]
				got := tc.sanitizer.Sanitize(in)
				assert.Equal(t, want, got)
			}
		})
	}
}

func TestWriter(t *testing.T) {
	s := &Sanitizer{
		Rules: makeRules(
			regexp.MustCompile(`[Hh](i|e(llo|y))`), "greeting",
			".", "!",
		),
	}

	in := "Hello, hi, hey there."
	out := s.Sanitize(in)
	require.Equal(t, out, "greeting, greeting, greeting there!")

	var buf bytes.Buffer
	_, err := s.Writer(&buf).Write([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, out, buf.String())
}

// makeRules converts each pair of args <pattern, replacer> into a rules map
// testing helper
func makeRules(args ...interface{}) []*Rule {
	if len(args)%2 != 0 {
		panic("makeRules requires an even number of args")
	}

	rules := make([]*Rule, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		var pattern *regexp.Regexp
		switch p := args[i].(type) {
		case *regexp.Regexp:
			pattern = p
		case regexp.Regexp:
			pattern = &p
		case string:
			pattern = regexp.MustCompile(regexp.QuoteMeta(p))
		default:
			panic(fmt.Sprintf("bad pattern type %T", args[i]))
		}

		var replacer ReplacerFunc
		switch r := args[i+1].(type) {
		case ReplacerFunc:
			replacer = r
		case func(string) string:
			replacer = r
		case string:
			replacer = func(string) string {
				return r
			}
		default:
			panic(fmt.Sprintf("bad replacer type %T", args[i]))
		}

		rules = append(rules, &Rule{
			Pattern:  pattern,
			Replacer: replacer,
		})
	}

	return rules
}

func counterReplacer() ReplacerFunc {
	var c int
	return func(s string) string {
		c++
		return fmt.Sprintf("<%d:%s>", c, s)
	}
}

func must(s *Sanitizer, err error) *Sanitizer {
	if err != nil {
		panic(err)
	}

	return s
}
