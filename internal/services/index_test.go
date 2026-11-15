package services

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/stretchr/testify/mock"

	"github.com/etkecc/mrs/internal/model"
)

func TestIndexRoomsBatchKeepsFlushTrigger(t *testing.T) {
	cfgMock := NewMockConfigService(t)
	cfgMock.EXPECT().Get().Return(&model.Config{
		Batch: &model.ConfigBatch{Rooms: 2},
	}).Maybe()

	indexMock := NewMockIndexRepository(t)
	mem, err := bleve.NewMemOnly(bleve.NewIndexMapping())
	if err != nil {
		t.Fatal(err)
	}
	indexMock.EXPECT().NewBatch().Return(mem.NewBatch()).Once()

	sizes := []int{}
	indexMock.EXPECT().IndexBatch(mock.Anything).Run(func(batch *bleve.Batch) {
		sizes = append(sizes, batch.Size())
	}).Return(nil).Twice()

	idx := NewIndex(cfgMock, indexMock)
	for i := range 3 {
		if err := idx.RoomsBatch(context.Background(), fmt.Sprintf("!room%d:example.org", i), &model.Entry{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.IndexBatch(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(sizes, []int{2, 1}) {
		t.Fatalf("batch sizes = %v, want [2 1]", sizes)
	}
}
