package services

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/etkecc/mrs/internal/model"
)

// each phase's CAS guard rejects re-entry while it's already running. no mock is wired, so any work the
// early-return skips would surface as an unexpected call and fail the test.
func TestPhaseGuards_RejectReentry(t *testing.T) {
	tests := []struct {
		name string
		flag func(m *Crawler) *atomic.Bool
		call func(m *Crawler)
	}{
		{"discovering", func(m *Crawler) *atomic.Bool { return &m.discovering }, func(m *Crawler) { m.DiscoverServers(context.Background(), 1) }},
		{"parsing", func(m *Crawler) *atomic.Bool { return &m.parsing }, func(m *Crawler) { m.ParseRooms(context.Background(), 1) }},
		{"eachrooming", func(m *Crawler) *atomic.Bool { return &m.eachrooming }, func(m *Crawler) {
			m.EachRoom(context.Background(), func(string, *model.MatrixRoom) bool { return false })
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewMockConfigService(t)
			fed := NewMockFederationService(t)
			v := NewMockValidatorService(t)
			block := NewMockBlocklistService(t)
			media := NewMockMediaService(t)
			data := NewMockDataRepository(t)

			m := NewCrawler(cfg, fed, v, block, media, data, nil)
			tt.flag(m).Store(true) // phase already in progress
			tt.call(m)
		})
	}
}

// harvested servers get persisted for next cycle, not dialed inline. the discovery mocks stay un-wired, so an inline dial fails the test.
func TestParseRooms_DefersHarvestNotDiscovers(t *testing.T) {
	cfg := NewMockConfigService(t)
	fed := NewMockFederationService(t)
	v := NewMockValidatorService(t)
	block := NewMockBlocklistService(t)
	media := NewMockMediaService(t)
	data := NewMockDataRepository(t)

	ctx := context.Background()
	cfg.EXPECT().Get().Return(&model.Config{Matrix: &model.ConfigMatrix{ServerName: "mrs.example"}}).Maybe()

	// one indexable server to parse
	indexable := map[string]*model.MatrixServer{
		"known.example": {Name: "known.example", Online: true, Indexable: true},
	}
	data.EXPECT().FilterServers(mock.Anything, mock.Anything).Return(indexable).Once()                        // IndexableServers
	data.EXPECT().FilterServers(mock.Anything, mock.Anything).Return(map[string]*model.MatrixServer{}).Once() // removeOldOfflineServers
	block.EXPECT().ByServer("known.example").Return(false)

	// its public rooms surface a brand-new server via the client directory.
	// the topic's (MRS-language:EN-MRS) directive pins the language so room.Parse never touches the nil detector.
	resp := &model.RoomDirectoryResponse{
		Chunk: []*model.RoomDirectoryRoom{
			{ID: "!r:known.example", Alias: "#r:known.example", Name: "Test", Topic: "(MRS-language:EN-MRS)", JoinRule: "public"},
		},
	}
	fed.EXPECT().QueryPublicRooms(mock.Anything, "known.example", mock.Anything, mock.Anything).Return(resp, nil)
	v.EXPECT().IsRoomAllowed(mock.Anything).Return(true)
	fed.EXPECT().QueryDirectoryExternal(mock.Anything, mock.Anything).Return(&model.QueryDirectoryResponse{Servers: []string{"new.example"}}, nil)
	data.EXPECT().AddRoomBatch(mock.Anything, mock.Anything).Return()
	data.EXPECT().AddRoomMapping(mock.Anything, mock.Anything, mock.Anything).Return(nil)
	data.EXPECT().FlushRoomBatch(mock.Anything).Return()

	// the assertion: the harvested server is handed to BatchServers (deferred persist), not discovered inline.
	var batched []string
	data.EXPECT().BatchServers(mock.Anything, mock.Anything).Run(func(_ context.Context, servers []string) {
		batched = servers
	}).Return(nil).Once()

	// afterRoomParsing tail: no rooms flow through the mocked EachRoom, so mapping/removal stay empty.
	data.EXPECT().EachRoom(mock.Anything, mock.Anything).Return()
	data.EXPECT().SetBiggestRooms(mock.Anything, mock.Anything).Return(nil)

	m := NewCrawler(cfg, fed, v, block, media, data, nil)
	m.ParseRooms(ctx, 1)

	if !slices.Contains(batched, "new.example") {
		t.Fatalf("BatchServers did not receive the harvested server; got %v", batched)
	}
}
