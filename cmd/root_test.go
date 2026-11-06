package cmd

import (
	"testing"

	cfg "register/pkg/config"

	"github.com/plaid/plaid-go/v15/plaid"
)

func TestLinkTokenToString(t *testing.T) {
	tests := []struct {
		name string
		tok  LinkToken
		want string
	}{
		{"value", LinkToken{LinkToken: "link-abc"}, "link-abc"},
		{"empty", LinkToken{}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok := tc.tok
			if got := tok.ToString(); got != tc.want {
				t.Errorf("LinkToken.ToString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAccessTokenToString(t *testing.T) {
	tests := []struct {
		name string
		tok  AccessToken
		want string
	}{
		{"value", AccessToken{AccessToken: "access-abc"}, "access-abc"},
		{"empty", AccessToken{}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok := tc.tok
			if got := tok.ToString(); got != tc.want {
				t.Errorf("AccessToken.ToString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetBankingClient(t *testing.T) {
	old := config
	defer func() { config = old }()

	t.Run("zero config builds a development client", func(t *testing.T) {
		config = &cfg.Config{}
		client := GetBankingClient()
		if client == nil {
			t.Fatal("GetBankingClient returned nil")
		}
		if client.Environment != plaid.Development {
			t.Errorf("Environment = %v, want Development", client.Environment)
		}
		if client.PlaidClient == nil {
			t.Error("PlaidClient is nil")
		}
	})

	t.Run("production config builds a production client with the prod secret", func(t *testing.T) {
		config = &cfg.Config{
			UserID:           "user-1",
			PlaidClientID:    "client-1",
			PlaidEnvironment: "production",
			PlaidEnvSecrets:  map[string]string{"production": "secret-prod"},
		}
		client := GetBankingClient()
		if client == nil {
			t.Fatal("GetBankingClient returned nil")
		}
		if client.Environment != plaid.Production {
			t.Errorf("Environment = %v, want Production", client.Environment)
		}
		if client.UserID != "user-1" {
			t.Errorf("UserID = %q, want %q", client.UserID, "user-1")
		}
		if client.ClientID != "client-1" {
			t.Errorf("ClientID = %q, want %q", client.ClientID, "client-1")
		}
		if client.Secret != "secret-prod" {
			t.Errorf("Secret = %q, want %q", client.Secret, "secret-prod")
		}
	})

	t.Run("non-production environment stays on development", func(t *testing.T) {
		config = &cfg.Config{PlaidEnvironment: "sandbox"}
		client := GetBankingClient()
		if client.Environment != plaid.Development {
			t.Errorf("Environment = %v, want Development", client.Environment)
		}
	})
}

// TestCobraHelpPaths exercises the command tree through the cobra harness
// without running any command: --help returns nil and never reaches a RunE,
// an unknown command returns an error, and neither path calls os.Exit
// (rootCmd.Execute is called directly, not the package-level Execute,
// which exits on error).
func TestCobraHelpPaths(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"update", "--help"},
		{"columns", "--help"},
		{"columns", "list", "--help"},
		{"budget", "--help"},
		{"monthly", "--help"},
		{"copy", "--help"},
		{"migrate", "--help"},
		{"recategorize", "--help"},
	} {
		t.Run(args[0]+" help", func(t *testing.T) {
			rootCmd.SetArgs(args)
			defer rootCmd.SetArgs(nil)
			var err error
			captureStdout(t, func() { err = rootCmd.Execute() })
			if err != nil {
				t.Errorf("rootCmd.Execute(%v) returned error: %v", args, err)
			}
		})
	}

	t.Run("unknown command returns an error", func(t *testing.T) {
		rootCmd.SetArgs([]string{"definitely-not-a-command"})
		defer rootCmd.SetArgs(nil)
		var err error
		captureStdout(t, func() { err = rootCmd.Execute() })
		if err == nil {
			t.Error("rootCmd.Execute(unknown command) returned nil error, want an error")
		}
	})
}
