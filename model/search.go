package model

// Entry represents indexable and/or indexed matrix room
type Entry struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Alias      string `json:"alias"`
	Name       string `json:"name"`
	Topic      string `json:"topic"`
	Avatar     string `json:"avatar"`
	AvatarURL  string `json:"avatar_url"`
	PreviewURL string `json:"preview_url"`
	Server     string `json:"server"`
	Members    int    `json:"members"`
	Language   string `json:"language"`
}
