package version

const (
	// Version of MRS
	Version = "v0.0.0"
	// Name of the project
	Name = "MatrixRoomsSearch"
)

var (
	// UserAgent of MRS
	UserAgent = Name + "/" + Version
	// Server header returned by MRS
	Server = Name + "/" + Version
)
