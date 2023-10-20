package model

// Entry represents indexable and/or indexed matrix room
type Entry struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Alias     string `json:"alias"`
	Name      string `json:"name"`
	Topic     string `json:"topic"`
	Avatar    string `json:"avatar"`
	AvatarURL string `json:"avatar_url"`
	Server    string `json:"server"`
	Members   int    `json:"members"`
	Language  string `json:"language"`
	// DEPRECATED
	PreviewURL string `json:"preview_url"`
}

// IsBlocked checks if room's server is blocked
func (r *Entry) IsBlocked(block BlocklistService) bool {
	if block.ByID(r.ID) {
		return true
	}
	if block.ByID(r.Alias) {
		return true
	}
	if block.ByServer(r.Server) {
		return true
	}
	return false
}
