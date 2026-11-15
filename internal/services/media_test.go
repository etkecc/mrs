package services

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/etkecc/mrs/internal/model"
)

func TestMediaGetShortFile(t *testing.T) {
	mediaDir := t.TempDir()
	cfgMock := NewMockConfigService(t)
	cfgMock.EXPECT().Get().Return(&model.Config{
		Public: &model.ConfigPublic{API: "https://example.org"},
		Path:   &model.ConfigPaths{Media: mediaDir},
	}).Maybe()

	m, err := NewMedia(cfgMock)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("\x89PNG\r\n\x1a\nsmall")
	path := m.getPath("server.example", "avatar", nil)
	if path == "" {
		t.Fatal("getPath returned an empty path")
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	content, contentType := m.Get(context.Background(), "server.example", "avatar", nil)
	if content == nil {
		t.Fatal("short media file was not served")
	}
	got, err := io.ReadAll(content)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("served bytes differ from the cached file")
	}
	if contentType != "image/png" {
		t.Fatalf("contentType = %q, want image/png", contentType)
	}

	htmlPath := m.getPath("html.example", "avatar", nil)
	if htmlPath == "" {
		t.Fatal("getPath returned an empty HTML path")
	}
	if err := os.WriteFile(htmlPath, []byte("<html><script>alert(1)</script></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	content, contentType = m.Get(context.Background(), "html.example", "avatar", nil)
	if content != nil || contentType != "" {
		t.Fatalf("non-image media served with type %q", contentType)
	}
}
