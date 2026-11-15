package model

import "testing"

func TestRoomDirectoryRoom_Convert(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		alias  string
		origin string
		want   string
	}{
		{"id server wins", "!room:room.example", "#room:alias.example", "found.example", "room.example"},
		{"alias server when id has none", "!abcdef123456", "#room:alias.example", "found.example", "alias.example"},
		{"origin when id and alias have none", "!abcdef123456", "", "found.example", "found.example"},
		{"empty id and alias use origin", "", "", "found.example", "found.example"},
		{"empty when no server is known", "!abcdef123456", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room := (&RoomDirectoryRoom{ID: tt.id, Alias: tt.alias}).Convert(tt.origin)
			if room.Server != tt.want {
				t.Errorf("Convert().Server = %q, want %q", room.Server, tt.want)
			}
		})
	}
}
