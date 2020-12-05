package config

type Options struct {
	SpreadsheetID string
	StartRow      int64
	EndRow        int64
	Copies        int
	Debug         bool
	Verbose       bool
	Test          bool
}

