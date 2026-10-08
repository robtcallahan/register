package config

type Options struct {
	SpreadsheetID string
	StartRow      int64
	EndRow        int64
	NumCopies     int
	BankIDs       []string
	Debug         bool
	Update        bool
	UseCSVFiles   bool
}
