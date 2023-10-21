package version

const (
	// Version of MRS
	Version = "v0.0.0"
	// Bot name for robots.txt
	Bot = "MRSBot"
)

var (
	// UserAgent of MRS
	UserAgent = "MatrixRoomsSearch/" + Version
	// Server header returned by MRS
	Server = "MatrixRoomsSearch/" + Version
)
