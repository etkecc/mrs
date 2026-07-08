package matrix

type matrixAuth struct {
	Origin      string
	Destination *string
	KeyID       string
	Signature   []byte
}
