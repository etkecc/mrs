package model

import (
	"slices"
	"testing"
)

func TestMatrixRoom_GetOwnServer(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		alias  string
		server string
		want   string
	}{
		{"id server wins", "!room:room.example", "#room:alias.example", "found.example", "room.example"},
		{"id server with port", "!room:room.example:8448", "#room:alias.example", "found.example", "room.example:8448"},
		{"alias server when id has none", "!abcdef123456", "#room:alias.example", "found.example", "alias.example"},
		{"stored server when id and alias have none", "!abcdef123456", "", "found.example", "found.example"},
		{"legacy comma list is ignored", "!abcdef123456", "", "old.example,other.example", ""},
		{"legacy comma list still yields id server", "!room:room.example", "", "old.example,other.example", "room.example"},
		{"legacy comma list still yields alias server", "!abcdef123456", "#room:alias.example", "old.example,other.example", "alias.example"},
		{"empty when no server is known", "!abcdef123456", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room := &MatrixRoom{ID: tt.id, Alias: tt.alias, Server: tt.server}
			if got := room.GetOwnServer(); got != tt.want {
				t.Errorf("GetOwnServer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMatrixRoom_EntryServer(t *testing.T) {
	room := &MatrixRoom{ID: "!room:room.example", Alias: "#room:alias.example", Server: "found.example"}
	if got := room.Entry().Server; got != "room.example" {
		t.Errorf("Entry().Server = %q, want %q", got, "room.example")
	}
}

func TestMatrixRoom_AllServers(t *testing.T) {
	tests := []struct {
		name string
		room *MatrixRoom
		want []string
	}{
		{
			"resolved server first, then participating servers",
			&MatrixRoom{ID: "!room:room.example", Alias: "#room:alias.example", Server: "found.example", Servers: []string{"extra.example"}},
			[]string{"room.example", "extra.example"},
		},
		{
			"resolved server deduplicated against participating servers",
			&MatrixRoom{ID: "!room:room.example", Servers: []string{"room.example", "extra.example"}},
			[]string{"room.example", "extra.example"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.room.AllServers(); !slices.Equal(got, tt.want) {
				t.Errorf("AllServers() = %v, want %v", got, tt.want)
			}
		})
	}
}
