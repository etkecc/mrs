package batch

import (
	"context"
	"sync"
	"testing"
)

func TestAddFlushesAtSize(t *testing.T) {
	var flushes [][]int
	b := New(3, func(_ context.Context, items []int) {
		flushes = append(flushes, append([]int(nil), items...))
	})

	for i := range 3 {
		b.Add(t.Context(), i)
	}

	if len(flushes) != 1 {
		t.Fatalf("expected 1 flush, got %d: %v", len(flushes), flushes)
	}
	if len(flushes[0]) != 3 {
		t.Fatalf("expected 3 items in flush, got %v", flushes[0])
	}
}

func TestAddConcurrentFlushesEveryItemOnce(t *testing.T) {
	const workers, perWorker = 8, 250
	total := workers * perWorker

	var mu sync.Mutex
	seen := map[int]int{}
	b := New(50, func(_ context.Context, items []int) {
		mu.Lock()
		defer mu.Unlock()
		for _, item := range items {
			seen[item]++
		}
	})

	ctx := t.Context()
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for i := range perWorker {
				b.Add(ctx, w*perWorker+i)
			}
		})
	}
	wg.Wait()
	b.Flush(ctx)

	for i := range total {
		if seen[i] != 1 {
			t.Fatalf("item %d flushed %d times", i, seen[i])
		}
	}
}
