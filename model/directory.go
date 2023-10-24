package model

// RoomDirectoryRequest sent when calling /_matrix/federation/v1/publicRooms
type RoomDirectoryRequest struct {
	Filter               RoomDirectoryFilter `json:"filter"`
	IncludeAllNetworks   bool                `query:"include_all_networks" json:"include_all_networks"`
	Limit                int                 `query:"limit" json:"limit"`
	Since                string              `query:"since" json:"since"`
	ThirdPartyInstanceID string              `query:"third_party_instance_id" json:"third_party_instance_id"`
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
	Avatar        string `json:"avatar_url"`
	Alias         string `json:"canonical_alias"`
	Guest         bool   `json:"guest_can_join"`
	Name          string `json:"name"`
	Members       int    `json:"num_joined_members"`
	ID            string `json:"room_id"`
	Topic         string `json:"topic"`
	WorldReadable bool   `json:"world_readable"`
}
