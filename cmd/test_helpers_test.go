package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// errTestBoom is the fake repo's stand-in failure.
var errTestBoom = errors.New("boom")

// captureStdout runs f with os.Stdout redirected to a pipe and returns
// everything f printed.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	f()
	if err := w.Close(); err != nil {
		t.Fatalf("closing captured stdout: %v", err)
	}
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return buf.String()
}

// newReader builds the bufio.Reader the stdin-driven helpers expect,
// backed by scripted input.
func newReader(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}
