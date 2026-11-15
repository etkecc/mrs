package services

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/etkecc/go-apm"
)

func TestConfigConcurrentReadGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("address: 127.0.0.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &Config{mu: &sync.Mutex{}, path: path}
	ctx := apm.NewContext()
	c.Read(ctx)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 10 {
			c.Read(ctx)
		}
	}()
	go func() {
		defer wg.Done()
		for range 10 {
			if c.Get() == nil {
				t.Error("config disappeared during reload")
			}
		}
	}()
	wg.Wait()
}
