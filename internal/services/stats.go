package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"

	"github.com/etkecc/go-apm/v2"
	"github.com/etkecc/mrs/internal/metrics"
	"github.com/etkecc/mrs/internal/model"
)

// webhookClient is fast-fail, retry-free: a webhook POST must never replay or hoard the federation pool.
var webhookClient = &http.Client{Timeout: 10 * time.Second}

type StatsRepository interface {
	DataRepository
	GetIndexStatsTL(ctx context.Context, prefix string) (map[time.Time]*model.IndexStats, error)
	SetIndexStatsTL(ctx context.Context, calculatedAt time.Time, stats *model.IndexStats) error
	GetIndexStats(ctx context.Context) *model.IndexStats
	SetIndexOnlineServers(ctx context.Context, servers int) error
	SetIndexIndexableServers(ctx context.Context, servers int) error
	SetIndexBlockedServers(ctx context.Context, servers int) error
	SetIndexServersSoftware(ctx context.Context, software map[string]int) error
	SetIndexParsedRooms(ctx context.Context, rooms int) error
	SetIndexIndexedRooms(ctx context.Context, rooms int) error
	SetIndexBannedRooms(ctx context.Context, rooms int) error
	SetIndexReportedRooms(ctx context.Context, rooms int) error
	SetStartedAt(ctx context.Context, process string, startedAt time.Time) error
	SetFinishedAt(ctx context.Context, process string, finishedAt time.Time) error
}

type Lenable interface {
	Len() int
}

// Stats service
type Stats struct {
	cfg        ConfigService
	data       StatsRepository
	block      Lenable
	index      Lenable
	mu         sync.RWMutex
	stats      *model.IndexStats
	collecting atomic.Bool
}

// softwareNameRE to normalize software names
var softwareNameRE = regexp.MustCompile(`[^a-z]+`)

// NewStats service
func NewStats(cfg ConfigService, data StatsRepository, index, blocklist Lenable) *Stats {
	stats := &Stats{cfg: cfg, data: data, index: index, block: blocklist}
	stats.reload(apm.NewContext())

	return stats
}

// setMetrics updates /metrics endpoint with actual stats
//
//nolint:gosec // no integer overflow here, as the counters are too low
func (s *Stats) setMetrics() {
	stats := s.Get()
	if stats == nil {
		return
	}
	metrics.ServersOnline.Set(uint64(stats.Servers.Online))
	metrics.ServersIndexable.Set(uint64(stats.Servers.Indexable))
	metrics.RoomsParsed.Set(uint64(stats.Rooms.Parsed))
	metrics.RoomsIndexed.Set(uint64(stats.Rooms.Indexed))
}

// reload saved stats. Useful when you need to get updated timestamps, but don't want to parse whole db
func (s *Stats) reload(ctx context.Context) {
	s.setStats(s.data.GetIndexStats(ctx))
	s.setMetrics()
}

// setStats replaces the cached stats snapshot under the lock.
func (s *Stats) setStats(stats *model.IndexStats) {
	s.mu.Lock()
	s.stats = stats
	s.mu.Unlock()
}

// Get stats
func (s *Stats) Get() *model.IndexStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// GetTL stats timeline
func (s *Stats) GetTL(ctx context.Context) map[time.Time]*model.IndexStats {
	tl, err := s.data.GetIndexStatsTL(ctx, "")
	if err != nil {
		apm.Log(ctx).Error("cannot get stats timeline", "error", err)
	}
	return tl
}

// SetStartedAt of the process
func (s *Stats) SetStartedAt(ctx context.Context, process string, startedAt time.Time) {
	if err := s.data.SetStartedAt(ctx, process, startedAt); err != nil {
		apm.Log(ctx).Error("cannot set started_at", "error", err, "process", process)
	}
	s.setStats(s.data.GetIndexStats(ctx))
}

// SetFinishedAt of the process
func (s *Stats) SetFinishedAt(ctx context.Context, process string, finishedAt time.Time) {
	if err := s.data.SetFinishedAt(ctx, process, finishedAt); err != nil {
		apm.Log(ctx).Error("cannot set finished_at", "error", err, "process", process)
	}
	s.setStats(s.data.GetIndexStats(ctx))
}

// CollectServers stats only
//
//nolint:gocognit // refactor software stats collection
func (s *Stats) CollectServers(ctx context.Context, reload bool) {
	var online, indexable, unknown int
	software := map[string]int{}
	s.data.FilterServers(ctx, func(server *model.MatrixServer) bool {
		if server.Online {
			online++
			name := strings.ToLower(server.Software)
			name = softwareNameRE.ReplaceAllString(name, " ")
			name = strings.Join(strings.Fields(name), " ")
			if name != "" {
				software[name]++
			} else {
				unknown++
			}

		}
		if server.Indexable {
			indexable++
		}
		return false
	})

	softwareThreshold := int(math.Ceil(float64(online) * 0.01)) // 1% or more
	belowThreshold := 0
	for name, count := range software {
		if count < softwareThreshold {
			belowThreshold += count
			delete(software, name)
		}
	}
	software["other"] = belowThreshold + unknown

	log := apm.Log(ctx)
	if err := s.data.SetIndexOnlineServers(ctx, online); err != nil {
		log.Error("cannot set online servers count", "error", err)
	}

	if err := s.data.SetIndexIndexableServers(ctx, indexable); err != nil {
		log.Error("cannot set indexable servers count", "error", err)
	}

	if err := s.data.SetIndexBlockedServers(ctx, s.block.Len()); err != nil {
		log.Error("cannot set blocked servers count", "error", err)
	}

	if err := s.data.SetIndexServersSoftware(ctx, software); err != nil {
		log.Error("cannot set servers software stats", "error", err)
	}

	if reload {
		s.reload(ctx)
	}
}

// Collect all stats from repository
func (s *Stats) Collect(ctx context.Context) {
	log := apm.Log(ctx)

	if !s.collecting.CompareAndSwap(false, true) {
		log.Info("stats collection already in progress, ignoring request")
		return
	}
	defer s.collecting.Store(false)

	s.CollectServers(ctx, false)

	var rooms int
	s.data.EachRoom(ctx, func(_ string, _ *model.MatrixRoom) bool {
		rooms++
		return false
	})
	if err := s.data.SetIndexParsedRooms(ctx, rooms); err != nil {
		log.Error("cannot set parsed rooms count", "error", err)
	}
	if err := s.data.SetIndexIndexedRooms(ctx, s.index.Len()); err != nil {
		log.Error("cannot set indexed rooms count", "error", err)
	}
	banned, berr := s.data.GetBannedRooms(ctx)
	if berr != nil {
		log.Error("cannot get banned rooms count", "error", berr)
	}
	if err := s.data.SetIndexBannedRooms(ctx, len(banned)); err != nil {
		log.Error("cannot set banned rooms count", "error", berr)
	}
	reported, rerr := s.data.GetReportedRooms(ctx)
	if rerr != nil {
		log.Error("cannot get reported rooms count", "error", berr)
	}
	if err := s.data.SetIndexReportedRooms(ctx, len(reported)); err != nil {
		log.Error("cannot set reported rooms count", "error", berr)
	}

	s.reload(ctx)
	if err := s.data.SetIndexStatsTL(ctx, time.Now().UTC(), s.Get()); err != nil {
		log.Error("cannot set stats timeline", "error", err)
	}
	s.sendWebhook(ctx)
}

// sendWebhook send request to webhook if provided
func (s *Stats) sendWebhook(ctx context.Context) {
	if s.cfg.Get().Webhooks.Stats == "" {
		return
	}
	log := apm.Log(ctx)

	var user string
	parsedUIURL, err := url.Parse(s.cfg.Get().Public.UI)
	if err == nil {
		user = parsedUIURL.Hostname()
	}

	payload, err := json.Marshal(webhookPayload{
		Username: user,
		Markdown: s.getWebhookText(),
	})
	if err != nil {
		log.Error("webhook payload marshaling failed", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.Get().Webhooks.Stats, bytes.NewReader(payload))
	if err != nil {
		log.Error("webhook request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := webhookClient.Do(req)
	if err != nil {
		log.Error("webhook sending failed", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		log.Error("webhook sending failed", "error", err, "status_code", resp.StatusCode, "body", string(body))
	}
}

func (s *Stats) getWebhookText() string {
	stats := s.Get()
	if stats == nil {
		return ""
	}
	var text strings.Builder
	text.WriteString("**stats have been collected**\n\n")

	fmt.Fprintf(&text, "* `%d` servers online (`%d` blocked)\n", stats.Servers.Online, stats.Servers.Blocked)
	fmt.Fprintf(&text, "* `%d` rooms (`%d` blocked, `%d` reported)\n", stats.Rooms.Indexed, stats.Rooms.Banned, stats.Rooms.Reported)
	text.WriteString("\n---\n\n")

	discovery := stats.Discovery.FinishedAt.Sub(stats.Discovery.StartedAt)
	parsing := stats.Parsing.FinishedAt.Sub(stats.Parsing.StartedAt)
	indexing := stats.Indexing.FinishedAt.Sub(stats.Indexing.StartedAt)
	total := discovery + parsing + indexing

	fmt.Fprintf(&text, "* `%s` took discovery process\n", discovery.String())
	fmt.Fprintf(&text, "* `%s` took parsing process\n", parsing.String())
	fmt.Fprintf(&text, "* `%s` took indexing process\n", indexing.String())
	fmt.Fprintf(&text, "* `%s` total\n", total.String())

	return text.String()
}
