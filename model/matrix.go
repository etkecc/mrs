package model

// MatrixRoom from matrix client-server API
type MatrixRoom struct {
	ID      string `json:"room_id"`
	Alias   string `json:"canonical_alias"`
	Name    string `json:"name"`
	Topic   string `json:"topic"`
	Avatar  string `json:"avatar_url"`
	Server  string `json:"server"` // custom
	Members int    `json:"num_joined_members"`
}

// Entry converts matrix room to search entry
func (r *MatrixRoom) Entry(server string) *Entry {
	if r.Server != "" {
		server = r.Server
	}
	return &Entry{
		ID:      r.ID,
		Type:    "room",
		Alias:   r.Alias,
		Name:    r.Name,
		Topic:   r.Topic,
		Avatar:  r.Avatar,
		Server:  server,
		Members: r.Members,
	}
}
