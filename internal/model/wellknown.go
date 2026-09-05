package model

// WellKnownClient is /.well-known/matrix/client: points a Matrix client at the homeserver (SRV never caught on).
type WellKnownClient struct {
	Homeserver WellKnownHomeserver `json:"m.homeserver"`
}

// WellKnownHomeserver is the m.homeserver block: just the client-server API base URL.
type WellKnownHomeserver struct {
	BaseURL string `json:"base_url"` // e.g. https://matrix.example.com
}

// WellKnownServer is /.well-known/matrix/server: federation delegation, points servers at our federation API.
type WellKnownServer struct {
	Host string `json:"m.server"` // host:port, e.g. matrix.example.com:443
}

// ClientVersions is /_matrix/client/versions: polite fiction, the spec-version list old clients and matrix.to expect.
type ClientVersions struct {
	Versions         []string        `json:"versions"`          // advertised client-server spec versions
	UnstableFeatures map[string]bool `json:"unstable_features"` // unstable MSCs we actually honor
}

// ServerVersion is /_matrix/federation/v1/version: our federation name and build. Boring, static, cached.
type ServerVersion struct {
	Server ServerVersionInfo `json:"server"`
}

// ServerVersionInfo is the name/version pair inside ServerVersion.
type ServerVersionInfo struct {
	Name    string `json:"name"`    // server software name
	Version string `json:"version"` // server software version
}

// RoomVisibility is directory/list/room/{roomID} body: always "public" (MRS holds only public; else 404, no body).
type RoomVisibility struct {
	Visibility string `json:"visibility"` // always "public"
}
