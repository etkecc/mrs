package utils

import "testing"

func TestIsValidAlias(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		alias string
		want  bool
	}{
		{"empty", "", false},
		{"valid simple", "#room:example.com", true},
		{"valid with port", "#room:example.com:8448", true},
		{"valid hyphen server", "#my-room:my-server.org", true},
		{"valid numeric server", "#room:1234.io", true},
		{"missing hash", "room:example.com", false},
		{"missing server", "#room:", false},
		{"missing colon", "#roomexample.com", false},
		{"double colon", "#room::example.com", false},
		{"invalid server chars", "#room:exa_mple.com", false},
		{"trailing slash", "#room:example.com/", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidAlias(tc.alias); got != tc.want {
				t.Errorf("IsValidAlias(%q) = %v, want %v", tc.alias, got, tc.want)
			}
		})
	}
}

func TestIsValidID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"empty", "", false},
		{"too short", "!short:s", false},
		{"valid room id", "!abcdefghijklmnop:srv.com", true},
		{"missing bang prefix", "abcdefghijklmnop:srv.com", false},
		{"hash prefix not bang", "#abcdefghijklmnop:srv.com", false},
		{"at boundary 11 chars", "!abcdefghij:s", true},
		{"at boundary 10 chars", "!abcdefg:s", false},
		{"too long 256 chars", "!" + string(make([]byte, 254)) + ":s", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidID(tc.id); got != tc.want {
				t.Errorf("IsValidID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

func TestServerFrom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		matrixID string
		want     string
	}{
		{"empty", "", ""},
		{"no colon", "!opaqueID", ""},
		{"trailing colon", "wrongid:", ""},
		{"room id", "!opaqueID:example.com", "example.com"},
		{"room alias", "#room:example.com", "example.com"},
		{"user id", "@user:example.com", "example.com"},
		{"with port", "!opaqueID:example.com:8448", "8448"},
		{"event id", "$event:example.com", "example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ServerFrom(tc.matrixID); got != tc.want {
				t.Errorf("ServerFrom(%q) = %q, want %q", tc.matrixID, got, tc.want)
			}
		})
	}
}
