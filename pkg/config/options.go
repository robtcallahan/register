package config

type Options struct {
	SpreadsheetID string
	StartRow      int64
	EndRow        int64
	Copies        int
	BankKeys      []string
	Debug         bool
	Verbose       bool
	Test          bool
}
