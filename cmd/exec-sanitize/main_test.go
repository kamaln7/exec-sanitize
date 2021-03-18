package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseArgs(t *testing.T) {
	tcs := []struct {
		args       []string
		wantParsed *parsedArgs
		wantErr    string
	}{
		{
			args:       []string{"--"},
			wantParsed: &parsedArgs{},
		},
		{
			args: []string{"--", "true"},
			wantParsed: &parsedArgs{
				cmd: "true",
			},
		},
		{
			args: []string{"--", "echo", "Hi, welcome to Chili's.", "Bye."},
			wantParsed: &parsedArgs{
				cmd:     "echo",
				cmdArgs: []string{"Hi, welcome to Chili's.", "Bye."},
			},
		},
		{
			args: []string{
				"-log", "/tmp",
				"-p:plain", "Hi", "-r", "Hello",
				"-p:plain", "^escape$", "-r", "1234",
				"-p:regex", "some pattern", "-r", "another",
				"--", "echo", "Hi, welcome to Chili's.", "Bye.",
			},
			wantParsed: &parsedArgs{
				rules: []parsedRule{
					{
						pattern:     "Hi",
						replacement: "Hello",
					},
					{
						pattern:     `\^escape\$`,
						replacement: "1234",
					},
					{
						pattern:     "some pattern",
						replacement: "another",
					},
				},
				cmd:     "echo",
				cmdArgs: []string{"Hi, welcome to Chili's.", "Bye."},
				logPath: "/tmp",
			},
		},
		{
			args: []string{
				"-flag",
			},
			wantErr: `unbalanced number of args`,
		},
		{
			args: []string{
				"-ye", "val",
			},
			wantErr: `unrecognized flag -ye`,
		},
		{
			args: []string{
				"-p:plain", "val",
				"-p:plain", "331",
			},
			wantErr: `pattern must be followed with a replacement`,
		},
		{
			args: []string{
				"-p:regex", "val",
				"-p:regex", "123",
			},
			wantErr: `pattern must be followed with a replacement`,
		},
		{
			args: []string{
				"-p:regex", "val", "-r", "rep",
				"-r", "1asd",
			},
			wantErr: `replacement must be directly preceeded by a pattern`,
		},
	}

	for _, tc := range tcs {
		t.Run("", func(t *testing.T) {
			parsed, err := parseArgs(tc.args)
			if tc.wantErr != "" {
				require.Equal(t, tc.wantErr, err.Error())
			} else {
				require.Nil(t, err)
			}
			assert.Equal(t, tc.wantParsed, parsed)
		})
	}
}

func Test_main(t *testing.T) {
	tcs := []struct {
		name    string
		args    []string
		stdin   io.Reader
		withLog bool
		expect  func(t *testing.T, stdout, stderr string, exitCode int, log map[string]string)
	}{
		{
			args: []string{
				"-p:plain", "Hi", "-r", "Hello",
				"--", "echo", "well Hi there!",
			},
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log map[string]string) {
				assert.Empty(t, stderr)
				assert.Zero(t, exitCode)
				assert.Empty(t, log)
				assert.Equal(t, "well Hello there!\n", stdout)
			},
		},
		{
			args: []string{
				"--", "bash", "-c", "exit 5",
			},
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log map[string]string) {
				assert.Equal(t, "\ncommand exited with code 5\n", stderr)
				assert.Equal(t, 5, exitCode)
				assert.Empty(t, log)
				assert.Empty(t, stdout)
			},
		},
		{
			args: []string{
				"-p:regex", "(Hi|Bye)", "-r", "Greetings",
				"-p:plain", "welcome to", "-r", "you have arrived at",
				"--", "cat", "-",
			},
			stdin: strings.NewReader("Hi, welcome to Chili's. Bye."),
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log map[string]string) {
				assert.Empty(t, stderr)
				assert.Zero(t, exitCode)
				assert.Empty(t, log)
				assert.Equal(t, "Greetings, you have arrived at Chili's. Greetings.", stdout)
			},
		},
		{
			args: []string{
				"-p:regex", "(Hi|Bye)", "-r", "<greeting-*>",
				"-p:plain", "welcome to", "-r", "you have arrived at",
				"--", "bash", "-c", `
					for i in $(seq 1 3); do
						IFS=$'\n' read -r line
						echo "$line"
					done
				`,
			},
			stdin: &steppedReader{steps: []string{
				"Hi, welcome to Chili's.\n",
				"Bye.\n",
				"Another.",
			}},
			withLog: true,
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log map[string]string) {
				assert.Empty(t, stderr)
				assert.Zero(t, exitCode)
				assert.Equal(t, "<greeting-0>, you have arrived at Chili's.\n<greeting-2>.\nAnother.\n", stdout)

				assert.Equal(t, map[string]string{
					"0": "Hi",
					"1": "welcome to",
					"2": "Bye",
				}, log)
			},
		},
		{
			args: []string{
				"-p:regex", "(Hi|Bye)", "-r", "@discard",
				"--", "bash", "-c", `
					for i in $(seq 1 3); do
						IFS=$'\n' read -r line
						echo "$line"
						sleep 0.1
					done
				`,
			},
			stdin: &steppedReader{steps: []string{
				"Hi, this should be discarded\n",
				"yeet yeet yeet yeet yeet yeet yeet yeet\n",
				"this should Bye be discarded\n",
			}},
			withLog: true,
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log map[string]string) {
				assert.Empty(t, stderr)
				assert.Zero(t, exitCode)
				assert.Equal(t, "yeet yeet yeet yeet yeet yeet yeet yeet\n", stdout)

				assert.Equal(t, map[string]string{
					"0": "Hi",
					"1": "Bye",
				}, log)
			},
		},
		{
			args: []string{
				"--", "echo", "-n", "Testing", "123",
			},
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log map[string]string) {
				assert.Empty(t, stderr)
				assert.Zero(t, exitCode)
				assert.Empty(t, log)
				assert.Equal(t, "Testing 123", stdout)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.args
			var logPath string
			if tc.withLog {
				var err error
				logPath, err = ioutil.TempDir("", "execsanitize")
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = os.RemoveAll(logPath)
				})

				args = append([]string{"-log", logPath}, args...)
			}

			var stdout, stderr bytes.Buffer
			exitCode := run(tc.stdin, &stdout, &stderr, append([]string{"/opt/execsanitize"}, args...))

			var log map[string]string
			if logPath != "" {
				log = make(map[string]string)
				err := filepath.Walk(logPath, func(path string, info os.FileInfo, err error) error {
					require.NoError(t, err)
					if info.IsDir() {
						return nil
					}

					content, err := ioutil.ReadFile(path)
					require.NoError(t, err)
					log[info.Name()] = string(content)
					return nil
				})
				require.NoError(t, err)
			}
			tc.expect(t, stdout.String(), stderr.String(), exitCode, log)
		})
	}
}

type steppedReader struct {
	steps []string
	step  int
}

func (r *steppedReader) Read(p []byte) (n int, err error) {
	if r.step >= len(r.steps) {
		return 0, io.ErrUnexpectedEOF
	}

	s := r.steps[r.step]
	r.step++
	n = copy(p, []byte(s))
	if r.step == len(r.steps) {
		err = io.EOF
	}
	return
}
