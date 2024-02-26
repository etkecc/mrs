package model

// RoomDirectoryRequestQuery sent when calling GET /_matrix/federation/v1/publicRooms
type RoomDirectoryRequestQuery struct {
	IncludeAllNetworks   string `query:"include_all_networks"`
	Limit                string `query:"limit" json:"limit"`
	Since                string `query:"since"`
	ThirdPartyInstanceID string `query:"third_party_instance_id"`
}

// RoomDirectoryRequest sent when calling POST /_matrix/federation/v1/publicRooms
type RoomDirectoryRequest struct {
	Filter               RoomDirectoryFilter `json:"filter"`
	IncludeAllNetworks   bool                `json:"include_all_networks"`
	Limit                int                 `json:"limit"`
	Since                string              `json:"since"`
	ThirdPartyInstanceID string              `json:"third_party_instance_id"`
}

// RoomDirectoryFilter for the RoomDirectoryRequest
type RoomDirectoryFilter struct {
	GenericSearchTerm string   `json:"generic_search_term"`
	RoomTypes         []string `json:"room_types,omitempty"`
}

// RoomDirectoryResponse of /_matrix/federation/v1/publicRooms
type RoomDirectoryResponse struct {
	Chunk     []*RoomDirectoryRoom `json:"chunk"`
	NextBatch string               `json:"next_batch"`
	PrevBatch string               `json:"prev_batch"`
	Total     int                  `json:"total_room_count_estimate"`
}

// RoomDirectoryRoom is MatrixRoom, but without any computed fields
type RoomDirectoryRoom struct {
	Avatar        string `json:"avatar_url,omitempty"`
	Alias         string `json:"canonical_alias,omitempty"`
	GuestJoinable bool   `json:"guest_can_join"`
	JoinRule      string `json:"join_rule,omitempty"`
	Name          string `json:"name,omitempty"`
	Members       int    `json:"num_joined_members"`
	ID            string `json:"room_id"`
	RoomType      string `json:"room_type,omitempty"`
	Topic         string `json:"topic,omitempty"`
	WorldReadable bool   `json:"world_readable"`
}

// Convert room directory's room to matrix room
func (r *RoomDirectoryRoom) Convert() *MatrixRoom {
	return &MatrixRoom{
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
