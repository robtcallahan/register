package cmd

import (
	cfg "register/pkg/config"
)

var (
	config *cfg.Config
	// SpreadsheetID ...
	SpreadsheetID string
	// StartRow ...
	StartRow int64
	// EndRow ...
	EndRow int64
	// Debug ...
	Debug bool
	// Test ...
	Test bool
	err  error
)
