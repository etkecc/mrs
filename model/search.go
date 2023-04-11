package model

// Entry represents indexable and/or indexed matrix room
type Entry struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Alias   string `json:"alias"`
	Name    string `json:"name"`
	Topic   string `json:"topic"`
	Avatar  string `json:"avatar"`
	Server  string `json:"server"`
	Members int    `json:"members"`
}
