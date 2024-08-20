package model

// Entry represents indexable and/or indexed matrix room
type Entry struct {
	ID            string `json:"id" yaml:"id"`
	Type          string `json:"type"`
	Alias         string `json:"alias" yaml:"alias"`
	Name          string `json:"name" yaml:"name"`
	Topic         string `json:"topic" yaml:"topic"`
	Avatar        string `json:"avatar" yaml:"avatar"`
	AvatarURL     string `json:"avatar_url" yaml:"avatar_url"`
	Server        string `json:"server" yaml:"server"`
	Members       int    `json:"members" yaml:"members"`
	Language      string `json:"language" yaml:"language"`
	RoomType      string `json:"room_type" yaml:"room_type"`
	JoinRule      string `json:"join_rule" yaml:"join_rule"`
	GuestJoinable bool   `json:"guest_can_join" yaml:"guest_can_join"`
	WorldReadable bool   `json:"world_readable" yaml:"world_readable"`
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

// RoomDirectory converts processed matrix room intro room directory's room
func (r *Entry) RoomDirectory() *RoomDirectoryRoom {
	return &RoomDirectoryRoom{
		ID:            r.ID,
		Alias:         r.Alias,
		Name:          r.Name,
		Topic:         r.Topic,
		Avatar:        r.Avatar,
		Members:       r.Members,
		RoomType:      r.RoomType,
		JoinRule:      r.JoinRule,
		GuestJoinable: r.GuestJoinable,
		WorldReadable: r.WorldReadable,
	}
}
