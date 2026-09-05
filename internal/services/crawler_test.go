package services

import (
	"context"
	"net/http"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/etkecc/mrs/internal/model"
)

// each phase's CAS guard rejects re-entry while running; no mock wired, so a skipped early-return fails the test.
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

// harvested servers persist for next cycle instead of dialing inline; discovery mocks stay un-wired.
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

	// its public rooms surface a new server; (MRS-language:EN-MRS) pins language, skipping the nil detector.
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

// the backoff schedule is the whole feature; an off-by-one at a 7d/14d boundary silently reshapes the dial curve.
func TestOfflineBackoff(t *testing.T) {
	day := 24 * time.Hour
	tests := []struct {
		age  time.Duration
		want time.Duration
	}{
		{-1 * day, 0},       // clock stepped back: floored, not a dial-every-run herd
		{6 * day, 0},        // fresh corpse: dial every run
		{7 * day, 4 * day},  // boundary into the 7-14d band
		{13 * day, 4 * day}, // still in the 7-14d band
		{14 * day, 7 * day}, // boundary into weekly
		{40 * day, 7 * day}, // weekly holds (>30d never reaches here in prod, prune deleted it)
	}
	for _, tt := range tests {
		if got := offlineBackoff(tt.age); got != tt.want {
			t.Errorf("offlineBackoff(%v) = %v, want %v", tt.age, got, tt.want)
		}
	}
}

// loadServers dials online/never-checked stubs always, offline throttled by backoff; RunAndReturn runs the predicate.
func TestLoadServers_BackoffFilter(t *testing.T) {
	cfg := NewMockConfigService(t)
	fed := NewMockFederationService(t)
	v := NewMockValidatorService(t)
	block := NewMockBlocklistService(t)
	media := NewMockMediaService(t)
	data := NewMockDataRepository(t)

	ctx := context.Background()
	cfg.EXPECT().Get().Return(&model.Config{}).Maybe() // no seed list, so the DB filter is the whole story

	now := time.Now().UTC()
	day := 24 * time.Hour
	candidates := map[string]*model.MatrixServer{
		// always dialed
		"online.example": {Name: "online.example", Online: true},
		// zero CheckedAt: never dialed, due (harvest / pre-backoff migration path)
		"stub.example": {Name: "stub.example"},
		// age 10d → 4d gap, only 1d since last check → skip
		"notdue.example": {Name: "notdue.example", OnlineAt: now.Add(-10 * day), CheckedAt: now.Add(-1 * day)},
		// age 10d → 4d gap, 5d since last check → dial
		"due.example": {Name: "due.example", OnlineAt: now.Add(-10 * day), CheckedAt: now.Add(-5 * day)},
	}
	data.EXPECT().FilterServers(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, filter func(*model.MatrixServer) bool) map[string]*model.MatrixServer {
			out := map[string]*model.MatrixServer{}
			for name, s := range candidates {
				if filter(s) {
					out[name] = s
				}
			}
			return out
		},
	).Once()

	m := NewCrawler(cfg, fed, v, block, media, data, nil)
	got := m.loadServers(ctx).Slice()

	for _, want := range []string{"online.example", "stub.example", "due.example"} {
		if !slices.Contains(got, want) {
			t.Errorf("loadServers dropped %q; got %v", want, got)
		}
	}
	if slices.Contains(got, "notdue.example") {
		t.Errorf("loadServers dialed a not-due offline server; got %v", got)
	}
}

// every dial that persists stamps CheckedAt, so backoff has a clock to read.
func TestDiscoverServer_StampsCheckedAt(t *testing.T) {
	cfg := NewMockConfigService(t)
	fed := NewMockFederationService(t)
	v := NewMockValidatorService(t)
	block := NewMockBlocklistService(t)
	media := NewMockMediaService(t)
	data := NewMockDataRepository(t)

	ctx := context.Background()
	block.EXPECT().ByServer("test.example").Return(false)
	v.EXPECT().IsOnline(mock.Anything, "test.example").Return("test.example", "Synapse", "1.0.0", true)
	fed.EXPECT().QueryCSURL(mock.Anything, "test.example").Return("https://test.example")
	v.EXPECT().IsIndexable(mock.Anything, "test.example").Return(true)

	var stored *model.MatrixServer
	data.EXPECT().AddServer(mock.Anything, mock.Anything).Run(func(_ context.Context, s *model.MatrixServer) {
		stored = s
	}).Return(nil).Once()

	m := NewCrawler(cfg, fed, v, block, media, data, nil)
	m.discoverServer(ctx, "test.example")

	if stored == nil {
		t.Fatal("discoverServer did not persist the server")
	}
	if stored.CheckedAt.IsZero() {
		t.Error("discoverServer left CheckedAt zero on an online dial")
	}
}

// clobber guard: resolves-but-down must skip AddServer (a Put would immortalize it); no mock wired, so a call fails.
func TestDiscoverServer_OfflineDoesNotPersist(t *testing.T) {
	cfg := NewMockConfigService(t)
	fed := NewMockFederationService(t)
	v := NewMockValidatorService(t)
	block := NewMockBlocklistService(t)
	media := NewMockMediaService(t)
	data := NewMockDataRepository(t)

	ctx := context.Background()
	block.EXPECT().ByServer("down.example").Return(false)
	v.EXPECT().IsOnline(mock.Anything, "down.example").Return("down.example", "Synapse", "1.0.0", false)

	m := NewCrawler(cfg, fed, v, block, media, data, nil)
	got := m.discoverServer(ctx, "down.example")

	if got.Online {
		t.Error("resolves-but-down server returned Online=true")
	}
	if got.Name != "down.example" {
		t.Errorf("offline return dropped the resolved name; got %q", got.Name)
	}
}

// AddServer's status tracks what persisted (online=201, offline=422); guards the old server==nil false-201 check.
func TestAddServer_StatusReflectsPersistence(t *testing.T) {
	t.Run("online persists and reports created", func(t *testing.T) {
		cfg := NewMockConfigService(t)
		fed := NewMockFederationService(t)
		v := NewMockValidatorService(t)
		block := NewMockBlocklistService(t)
		media := NewMockMediaService(t)
		data := NewMockDataRepository(t)

		ctx := context.Background()
		data.EXPECT().HasServer(mock.Anything, "up.example").Return(false)
		block.EXPECT().ByServer("up.example").Return(false)
		v.EXPECT().IsOnline(mock.Anything, "up.example").Return("up.example", "Synapse", "1.0.0", true)
		fed.EXPECT().QueryCSURL(mock.Anything, "up.example").Return("https://up.example")
		v.EXPECT().IsIndexable(mock.Anything, "up.example").Return(true)
		data.EXPECT().AddServer(mock.Anything, mock.Anything).Return(nil).Once()

		m := NewCrawler(cfg, fed, v, block, media, data, nil)
		if code := m.AddServer(ctx, "up.example"); code != http.StatusCreated {
			t.Errorf("online add: got %d, want %d", code, http.StatusCreated)
		}
	})

	t.Run("offline records a stub via the safe writer and reports unprocessable", func(t *testing.T) {
		cfg := NewMockConfigService(t)
		fed := NewMockFederationService(t)
		v := NewMockValidatorService(t)
		block := NewMockBlocklistService(t)
		media := NewMockMediaService(t)
		data := NewMockDataRepository(t)

		ctx := context.Background()
		data.EXPECT().HasServer(mock.Anything, "down.example").Return(false)
		block.EXPECT().ByServer("down.example").Return(false)
		v.EXPECT().IsOnline(mock.Anything, "down.example").Return("down.example", "", "", false)
		// the row stops the anonymous re-dial hammer via MarkServersOffline (safe writer); AddServer would clobber it.
		data.EXPECT().MarkServersOffline(mock.Anything, []string{"down.example"}).Return().Once()

		m := NewCrawler(cfg, fed, v, block, media, data, nil)
		if code := m.AddServer(ctx, "down.example"); code != http.StatusUnprocessableEntity {
			t.Errorf("offline add: got %d, want %d", code, http.StatusUnprocessableEntity)
		}
	})

	// once a name is known, a repeat POST short-circuits on HasServer; no re-dial mocks wired, so a re-dial fails.
	t.Run("already-known short-circuits without re-dialing", func(t *testing.T) {
		cfg := NewMockConfigService(t)
		fed := NewMockFederationService(t)
		v := NewMockValidatorService(t)
		block := NewMockBlocklistService(t)
		media := NewMockMediaService(t)
		data := NewMockDataRepository(t)

		ctx := context.Background()
		data.EXPECT().HasServer(mock.Anything, "known.example").Return(true)

		m := NewCrawler(cfg, fed, v, block, media, data, nil)
		if code := m.AddServer(ctx, "known.example"); code != http.StatusAlreadyReported {
			t.Errorf("known add: got %d, want %d", code, http.StatusAlreadyReported)
		}
	})
}
