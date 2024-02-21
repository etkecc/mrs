package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog"

	"gitlab.com/etke.cc/mrs/api/metrics"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type StatsRepository interface {
	DataRepository
	GetIndexStatsTL(ctx context.Context, prefix string) (map[time.Time]*model.IndexStats, error)
	SetIndexStatsTL(ctx context.Context, calculatedAt time.Time, stats *model.IndexStats) error
	GetIndexStats(ctx context.Context) *model.IndexStats
	SetIndexOnlineServers(ctx context.Context, servers int) error
	SetIndexIndexableServers(ctx context.Context, servers int) error
	SetIndexBlockedServers(ctx context.Context, servers int) error
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
	stats      *model.IndexStats
	collecting bool
}

// NewStats service
func NewStats(cfg ConfigService, data StatsRepository, index, blocklist Lenable) *Stats {
	stats := &Stats{cfg: cfg, data: data, index: index, block: blocklist}
	stats.reload(utils.NewContext())

	return stats
}

// setMetrics updates /metrics endpoint with actual stats
func (s *Stats) setMetrics() {
	metrics.ServersOnline.Set(uint64(s.stats.Servers.Online))
	metrics.ServersIndexable.Set(uint64(s.stats.Servers.Indexable))
	metrics.RoomsParsed.Set(uint64(s.stats.Rooms.Parsed))
	metrics.RoomsIndexed.Set(uint64(s.stats.Rooms.Indexed))
}

// reload saved stats. Useful when you need to get updated timestamps, but don't want to parse whole db
func (s *Stats) reload(ctx context.Context) {
	s.stats = s.data.GetIndexStats(ctx)
	s.setMetrics()
}

// Get stats
func (s *Stats) Get() *model.IndexStats {
	return s.stats
}

// GetTL stats timeline
func (s *Stats) GetTL(ctx context.Context) map[time.Time]*model.IndexStats {
	tl, err := s.data.GetIndexStatsTL(ctx, "")
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("cannot get stats timeline")
	}
	return tl
}

// SetStartedAt of the process
func (s *Stats) SetStartedAt(ctx context.Context, process string, startedAt time.Time) {
	if err := s.data.SetStartedAt(ctx, process, startedAt); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("process", process).Msg("cannot set started_at")
	}
	s.stats = s.data.GetIndexStats(ctx)
}

// SetFinishedAt of the process
func (s *Stats) SetFinishedAt(ctx context.Context, process string, finishedAt time.Time) {
	if err := s.data.SetFinishedAt(ctx, process, finishedAt); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("process", process).Msg("cannot set finished_at")
	}
	s.stats = s.data.GetIndexStats(ctx)
}

// CollectServers stats only
func (s *Stats) CollectServers(ctx context.Context, reload bool) {
	var online, indexable int
	s.data.FilterServers(ctx, func(server *model.MatrixServer) bool {
		if server.Online {
			online++
		}
		if server.Indexable {
			indexable++
		}
		return false
	})

	log := zerolog.Ctx(ctx)
	if err := s.data.SetIndexOnlineServers(ctx, online); err != nil {
		log.Error().Err(err).Msg("cannot set online servers count")
	}

	if err := s.data.SetIndexIndexableServers(ctx, indexable); err != nil {
		log.Error().Err(err).Msg("cannot set indexable servers count")
	}

	if err := s.data.SetIndexBlockedServers(ctx, s.block.Len()); err != nil {
		log.Error().Err(err).Msg("cannot set blocked servers count")
	}

	if reload {
		s.reload(ctx)
	}
}

// Collect all stats from repository
func (s *Stats) Collect(ctx context.Context) {
	span := utils.StartSpan(ctx, "stats.Collect")
	defer span.Finish()

	log := zerolog.Ctx(ctx)

	if s.collecting {
		log.Info().Msg("stats collection already in progress, ignoring request")
		return
	}
	s.collecting = true
	defer func() { s.collecting = false }()

	s.CollectServers(span.Context(), false)

	var rooms int
	s.data.EachRoom(span.Context(), func(_ string, _ *model.MatrixRoom) bool {
		rooms++
		return false
	})
	if err := s.data.SetIndexParsedRooms(span.Context(), rooms); err != nil {
		log.Error().Err(err).Msg("cannot set parsed rooms count")
	}
	if err := s.data.SetIndexIndexedRooms(span.Context(), s.index.Len()); err != nil {
		log.Error().Err(err).Msg("cannot set indexed rooms count")
	}
	banned, berr := s.data.GetBannedRooms(span.Context())
	if berr != nil {
		log.Error().Err(berr).Msg("cannot get banned rooms count")
	}
	if err := s.data.SetIndexBannedRooms(span.Context(), len(banned)); err != nil {
		log.Error().Err(berr).Msg("cannot set banned rooms count")
	}
	reported, rerr := s.data.GetReportedRooms(span.Context())
	if rerr != nil {
		log.Error().Err(berr).Msg("cannot get reported rooms count")
	}
	if err := s.data.SetIndexReportedRooms(span.Context(), len(reported)); err != nil {
		log.Error().Err(berr).Msg("cannot set reported rooms count")
	}

	s.reload(span.Context())
	if err := s.data.SetIndexStatsTL(span.Context(), time.Now().UTC(), s.stats); err != nil {
		log.Error().Err(err).Msg("cannot set stats timeline")
	}
	s.sendWebhook(span.Context())
}

// sendWebhook send request to webhook if provided
func (s *Stats) sendWebhook(ctx context.Context) {
	if s.cfg.Get().Webhooks.Stats == "" {
		return
	}
	span := utils.StartSpan(ctx, "stats.sendWebhook")
	defer span.Finish()
	log := zerolog.Ctx(ctx)

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
		log.Error().Err(err).Msg("webhook payload marshaling failed")
		return
	}

	req, err := http.NewRequest("POST", s.cfg.Get().Webhooks.Stats, bytes.NewReader(payload))
	if err != nil {
		log.Error().Err(err).Msg("webhook request failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("webhook sending failed")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		log.Error().Err(err).Int("status_code", resp.StatusCode).Str("body", string(body)).Msg("webhook sending failed")
	}
}

func (s *Stats) getWebhookText() string {
	var text strings.Builder
	text.WriteString("**stats have been collected**\n\n")

	text.WriteString(fmt.Sprintf("* `%d` servers online (`%d` blocked)\n", s.stats.Servers.Online, s.stats.Servers.Blocked))
	text.WriteString(fmt.Sprintf("* `%d` rooms (`%d` blocked, `%d` reported)\n", s.stats.Rooms.Indexed, s.stats.Rooms.Banned, s.stats.Rooms.Reported))
	text.WriteString("\n---\n\n")

	discovery := s.stats.Discovery.FinishedAt.Sub(s.stats.Discovery.StartedAt)
	parsing := s.stats.Parsing.FinishedAt.Sub(s.stats.Parsing.StartedAt)
	indexing := s.stats.Indexing.FinishedAt.Sub(s.stats.Indexing.StartedAt)
	total := discovery + parsing + indexing

	text.WriteString(fmt.Sprintf("* `%s` took discovery process\n", discovery.String()))
	text.WriteString(fmt.Sprintf("* `%s` took parsing process\n", parsing.String()))
	text.WriteString(fmt.Sprintf("* `%s` took indexing process\n", indexing.String()))
	text.WriteString(fmt.Sprintf("* `%s` total\n", total.String()))

	return text.String()
}
