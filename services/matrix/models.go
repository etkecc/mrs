package matrix

// matrixKeyResp is response of /_matrix/key/v2/server
type matrixKeyResp struct {
	ServerName    string                       `json:"server_name"`
	ValidUntilTS  int64                        `json:"valid_until_ts"`
	VerifyKeys    map[string]map[string]string `json:"verify_keys"`
	OldVerifyKeys map[string]map[string]any    `json:"old_verify_keys"`
	Signatures    map[string]map[string]string `json:"signatures,omitempty"`
}

type wellKnownServerResp struct {
	Host string `json:"m.server"`
}

type wellKnownClientResp struct {
	Homeserver wellKnownClientRespHomeserver `json:"m.homeserver"`
}

type wellKnownClientRespHomeserver struct {
	BaseURL string `json:"base_url"`
}

type serverVersionResp struct {
	Server map[string]string `json:"server"`
}

type clientVersionResp struct {
	Versions         []string        `json:"versions"`
	UnstableFeatures map[string]bool `json:"unstable_features"`
}

type queryDirectoryResp struct {
	RoomID  string   `json:"room_id"`
	Servers []string `json:"servers"`
}

type matrixAuth struct {
	Origin      string
	Destination string
	KeyID       string
	Signature   []byte
}
