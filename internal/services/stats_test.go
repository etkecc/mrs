package services

import (
	"context"
	"sync"
	"testing"

	"github.com/etkecc/mrs/internal/model"
)

type statsRepoStub struct {
	StatsRepository
	stats *model.IndexStats
}

func (s *statsRepoStub) GetIndexStats(context.Context) *model.IndexStats {
	return s.stats
}

func TestStatsConcurrentGetReload(t *testing.T) {
	s := &Stats{
		data:  &statsRepoStub{stats: &model.IndexStats{}},
		stats: &model.IndexStats{},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 20 {
			s.reload(context.Background())
		}
	}()
	go func() {
		defer wg.Done()
		for range 20 {
			if s.Get() == nil {
				t.Error("stats disappeared during reload")
			}
		}
	}()
	wg.Wait()
}
