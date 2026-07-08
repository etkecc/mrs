package model

// WellKnownClient is /.well-known/matrix/client: the breadcrumb a Matrix client
// follows to find the homeserver, because SRV records never quite caught on with clients.
type WellKnownClient struct {
	Homeserver WellKnownHomeserver `json:"m.homeserver"`
}

// WellKnownHomeserver is the m.homeserver block: just the client-server API base URL.
type WellKnownHomeserver struct {
	BaseURL string `json:"base_url"` // e.g. https://matrix.example.com
}

// WellKnownServer is /.well-known/matrix/server: federation delegation, one line.
// Points other servers at where our federation API actually answers.
type WellKnownServer struct {
	Host string `json:"m.server"` // host:port, e.g. matrix.example.com:443
}

// ClientVersions is /_matrix/client/versions. We are a search index wearing a homeserver's
// coat, so most of this is polite fiction: a spec-version list old clients and matrix.to
// insist on seeing before they will talk to us.
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

// RoomVisibility is the 200 body of /_matrix/client/v3/directory/list/room/{roomID}.
// MRS holds only public rooms, so whenever this is returned Visibility is "public"; a room we do not hold or have banned is a 404 with no body.
type RoomVisibility struct {
	Visibility string `json:"visibility"` // always "public"
}
