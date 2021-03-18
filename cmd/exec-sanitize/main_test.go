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
			args: []string{"--"},
			wantParsed: &parsedArgs{
				rules: map[string]string{},
			},
		},
		{
			args: []string{"--", "true"},
			wantParsed: &parsedArgs{
				rules: map[string]string{},
				cmd:   "true",
			},
		},
		{
			args: []string{"--", "echo", "Hi, welcome to Chili's.", "Bye."},
			wantParsed: &parsedArgs{
				rules:   map[string]string{},
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
				rules: map[string]string{
					"Hi":           "Hello",
					`\^escape\$`:   "1234",
					"some pattern": "another",
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
		stdin   string
		withLog bool
		expect  func(t *testing.T, stdout, stderr string, exitCode int, log []string)
	}{
		{
			args: []string{
				"es", "-p:plain", "Hi", "-r", "Hello",
				"--", "echo", "well Hi there!",
			},
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log []string) {
				require.Empty(t, stderr)
				require.Zero(t, exitCode)
				require.Empty(t, log)
				require.Equal(t, "well Hello there!\n", stdout)
			},
		},
		{
			args: []string{
				"es",
				"--", "bash", "-c", "exit 5",
			},
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log []string) {
				require.Equal(t, "\ncommand exited with code 5\n", stderr)
				require.Equal(t, 5, exitCode)
				require.Empty(t, log)
				require.Empty(t, stdout)
			},
		},
		{
			args: []string{
				"es",
				"-p:regex", "(Hi|Bye)", "-r", "Greetings",
				"-p:plain", "welcome to", "-r", "you have arrived at",
				"--", "cat", "-",
			},
			stdin: "Hi, welcome to Chili's. Bye.",
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log []string) {
				require.Empty(t, stderr)
				require.Zero(t, exitCode)
				require.Empty(t, log)
				require.Equal(t, "Greetings, you have arrived at Chili's. Greetings.", stdout)
			},
		},
		{
			args: []string{
				"es",
				"--", "echo", "-n", "Testing", "123",
			},
			expect: func(t *testing.T, stdout, stderr string, exitCode int, log []string) {
				require.Empty(t, stderr)
				require.Zero(t, exitCode)
				require.Empty(t, log)
				require.Equal(t, "Testing 123", stdout)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.args
			var logPath string
			if tc.withLog {
				logPath, err := ioutil.TempDir("", "execsanitize")
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = os.RemoveAll(logPath)
				})

				args = append([]string{"-log", logPath}, args...)
			}

			var (
				stdin          io.Reader
				stdout, stderr bytes.Buffer
			)
			if tc.stdin != "" {
				stdin = strings.NewReader(tc.stdin)
			}
			exitCode := run(stdin, &stdout, &stderr, args)

			var log []string
			if logPath != "" {
				err := filepath.Walk(logPath, func(path string, info os.FileInfo, err error) error {
					require.NoError(t, err)
					if info.IsDir() {
						return nil
					}

					content, err := ioutil.ReadFile(path)
					require.NoError(t, err)
					log = append(log, string(content))
					return nil
				})
				require.NoError(t, err)
			}
			tc.expect(t, stdout.String(), stderr.String(), exitCode, log)
		})
	}
}
