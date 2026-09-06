package client_test

import (
	"testing"

	"envious-cli/internal/client"
)

func TestNewValidation(t *testing.T) {
	cases := []struct {
		name    string
		base    string
		key     string
		wantErr bool
	}{
		{"valid http", "http://127.0.0.1:8080", "k", false},
		{"valid https", "https://example.com", "k", false},
		{"empty base", "", "k", true},
		{"missing scheme", "127.0.0.1:8080", "k", true},
		{"missing host", "http://", "k", true},
		{"empty key", "http://127.0.0.1:8080", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.New(tc.base, tc.key)
			if tc.wantErr && err == nil {
				t.Fatalf("New(%q) = nil, want error", tc.base)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("New(%q) = %v, want nil", tc.base, err)
			}
		})
	}
}
