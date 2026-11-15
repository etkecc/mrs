package matrix

import (
	"bytes"
	"testing"
)

func TestReaderBytesRejectsOversize(t *testing.T) {
	s := &Server{}
	payload := make([]byte, maxThumbnailBytes+1)
	reader, contents := s.readerBytes(bytes.NewReader(payload))
	if reader != nil || contents != nil {
		t.Fatal("oversized avatar body was buffered")
	}
}

func TestReaderBytesAcceptsSmallBody(t *testing.T) {
	s := &Server{}
	payload := []byte("small")
	reader, contents := s.readerBytes(bytes.NewReader(payload))
	if reader == nil || !bytes.Equal(contents, payload) {
		t.Fatal("small avatar body was rejected")
	}
}
