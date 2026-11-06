package main

import (
	"os"
	"testing"
)

// TestMainHelpFlag exercises main() through the one argument path that
// provably returns: cobra returns flag.ErrHelp for --help before it ever
// calls the root command's Run (which os.Exit(0)s on no args), and
// Execute converts that into a nil return, so cmd.Execute() never reaches
// os.Exit(1) either. No other main() path is safe to call from a test:
// with no args Run calls os.Exit(0), and any execution error makes
// Execute call os.Exit(1) — either would kill the test binary.
func TestMainHelpFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"register", "--help"}
	main()
}
