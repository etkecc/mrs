package services

import (
	"sync"
	"testing"

	"github.com/etkecc/mrs/internal/model"
)

func TestBlocklistInvalidRegexSkipsEntry(t *testing.T) {
	cfgMock := NewMockConfigService(t)
	cfgMock.EXPECT().Get().Return(&model.Config{
		Blocklist: &model.ConfigBlocklist{Servers: []string{"("}},
	}).Maybe()

	b := NewBlocklist(cfgMock)
	if b.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", b.Len())
	}
	if b.ByServer("example.org") {
		t.Fatal("invalid regex matched a server")
	}
}

func TestBlocklistConcurrentAddAndRead(t *testing.T) {
	cfgMock := NewMockConfigService(t)
	cfgMock.EXPECT().Get().Return(&model.Config{
		Blocklist: &model.ConfigBlocklist{Servers: []string{`^blocked\.`}},
	}).Maybe()

	b := NewBlocklist(cfgMock)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			b.Add("dynamic.example")
		}()
		go func() {
			defer wg.Done()
			_ = b.ByServer("dynamic.example")
			_ = b.Len()
		}()
	}
	wg.Wait()

	if !b.ByServer("dynamic.example") {
		t.Fatal("dynamic blocklist entry was lost")
	}
	if b.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", b.Len())
	}
}
