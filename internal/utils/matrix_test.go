package utils

import (
	"strings"
	"testing"
)

func TestIsValidID(t *testing.T) {
	// server-name examples come from the spec appendices
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"domain form", "!opaque:matrix.org", true},
		{"short id", "!a:b", true},
		{"domain-less v12 id", "!abc123def456", true},
		{"with port", "!room:matrix.org:8888", true},
		{"IPv4 literal with port", "!room:1.2.3.4:1234", true},
		{"IPv6 literal", "!room:[1234:5678::abcd]", true},
		{"IPv6 literal with port", "!room:[1234:5678::abcd]:5678", true},
		{"unicode localpart", "!héllo:matrix.org", true},
		{"exactly 255 bytes", "!" + strings.Repeat("a", 252) + ":b", true},
		{"empty", "", false},
		{"sigil only", "!", false},
		{"empty localpart", "!:matrix.org", false},
		{"empty domain", "!room:", false},
		{"wrong sigil", "#room:matrix.org", false},
		{"no sigil", "room:matrix.org", false},
		{"NUL in localpart", "!ro\x00m:matrix.org", false},
		{"non-digit port", "!room:matrix.org:abc", false},
		{"empty port", "!room:matrix.org:", false},
		{"double port", "!room:server:8448:9999", false},
		{"unterminated bracket", "!room:[1234", false},
		{"text after bracket", "!room:[1234]def", false},
		{"empty bracket", "!room:[]", false},
		{"256 bytes", "!" + strings.Repeat("a", 253) + ":b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidID(tt.id); got != tt.want {
				t.Errorf("IsValidID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestIsValidAlias(t *testing.T) {
	tests := []struct {
		name  string
		alias string
		want  bool
	}{
		{"valid", "#room:matrix.org", true},
		{"with port", "#room:matrix.org:8888", true},
		{"IPv4 literal with port", "#room:1.2.3.4:1234", true},
		{"IPv6 literal", "#room:[1234:5678::abcd]", true},
		{"IPv6 literal with port", "#room:[1234:5678::abcd]:5678", true},
		{"unicode localpart", "#héllo:matrix.org", true},
		{"exactly 255 bytes", "#" + strings.Repeat("a", 252) + ":b", true},
		{"empty", "", false},
		{"no domain", "#room", false},
		{"sigil only", "#", false},
		{"empty localpart", "#:matrix.org", false},
		{"empty domain", "#room:", false},
		{"wrong sigil", "!room:matrix.org", false},
		{"no sigil", "room:matrix.org", false},
		{"NUL in localpart", "#ro\x00m:matrix.org", false},
		{"non-digit port", "#room:matrix.org:abc", false},
		{"unterminated bracket", "#room:[1234", false},
		{"256 bytes", "#" + strings.Repeat("a", 253) + ":b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidAlias(tt.alias); got != tt.want {
				t.Errorf("IsValidAlias(%q) = %v, want %v", tt.alias, got, tt.want)
			}
		})
	}
}

func TestServerFrom(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"empty", "", ""},
		{"no colon", "!opaque", ""},
		{"domain-less v12 id", "!abc123def456", ""},
		{"domain-less event id", "$abcdef123456", ""},
		{"user id", "@user:matrix.org", "matrix.org"},
		{"room id", "!room:matrix.org", "matrix.org"},
		{"room alias", "#room:matrix.org", "matrix.org"},
		{"room id with port", "!room:server:8448", "server:8448"},
		{"user id with port", "@someone:server:8448", "server:8448"},
		{"room alias with port", "#room:server:8448", "server:8448"},
		{"IPv6 literal", "!room:[1234:5678::abcd]", "[1234:5678::abcd]"},
		{"IPv6 literal with port", "!room:[1234:5678::abcd]:5678", "[1234:5678::abcd]:5678"},
		{"trailing colon", "@user:", ""},
		{"leading colon", ":server", "server"},
		{"plain host and port", "foo:bar", "bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServerFrom(tt.id); got != tt.want {
				t.Errorf("ServerFrom(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
