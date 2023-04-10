package model

// Entry represents indexable and/or indexed data
type Entry struct {
	ID      string `json:"id"`
	Alias   string `json:"alias"`
	Name    string `json:"name"`
	Topic   string `json:"topic"`
	Avatar  string `json:"avatar"`
	Members int    `json:"members"`
}
