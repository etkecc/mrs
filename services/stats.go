package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/exp/constraints"

	"gitlab.com/etke.cc/mrs/api/metrics"
	"gitlab.com/etke.cc/mrs/api/model"
)

type StatsRepository interface {
	DataRepository
	GetIndexStats() *model.IndexStats
	SetIndexOnlineServers(servers int) error
	SetIndexIndexableServers(servers int) error
	SetIndexBlockedServers(servers int) error
	SetIndexParsedRooms(rooms int) error
	SetIndexIndexedRooms(rooms int) error
	SetIndexBannedRooms(rooms int) error
	SetIndexReportedRooms(rooms int) error
	SetStartedAt(process string, startedAt time.Time) error
	SetFinishedAt(process string, finishedAt time.Time) error
}

type Lenable interface {
	Len() int
}

// Stats service
type Stats struct {
	data        StatsRepository
	block       Lenable
	index       Lenable
	stats       *model.IndexStats
	prev        *model.IndexStats
	webhookUser string
	webhook     string
	collecting  bool
}

// NewStats service
func NewStats(data StatsRepository, index, blocklist Lenable, uiurl, webhook string) *Stats {
	if uiurl != "" {
		parsedUIURL, err := url.Parse(uiurl)
		if err == nil {
			uiurl = parsedUIURL.Hostname()
		}
	}
	stats := &Stats{data: data, index: index, block: blocklist, webhook: webhook, webhookUser: uiurl}
	stats.reload()

	return stats
}

// setMetrics updates /metrics endpoint with actual stats
func (s *Stats) setMetrics() {
	metrics.ServersOnline.Set(float64(s.stats.Servers.Online))
	metrics.ServersIndexable.Set(float64(s.stats.Servers.Indexable))
	metrics.RoomsParsed.Set(float64(s.stats.Rooms.Parsed))
	metrics.RoomsIndexed.Set(float64(s.stats.Rooms.Indexed))
}

// reload saved stats. Useful when you need to get updated timestamps, but don't want to parse whole db
func (s *Stats) reload() {
	s.prev = s.stats.Clone()
	s.stats = s.data.GetIndexStats()
	s.setMetrics()
}

// Get stats
func (s *Stats) Get() *model.IndexStats {
	return s.stats
}

// SetStartedAt of the process
func (s *Stats) SetStartedAt(process string, startedAt time.Time) {
	if err := s.data.SetStartedAt(process, startedAt); err != nil {
		log.Println("cannot set", process, "started_at", err)
	}
	s.stats = s.data.GetIndexStats()
}

// SetFinishedAt of the process
func (s *Stats) SetFinishedAt(process string, finishedAt time.Time) {
	if err := s.data.SetFinishedAt(process, finishedAt); err != nil {
		log.Println("cannot set", process, "finished_at", err)
	}
	s.stats = s.data.GetIndexStats()
}

// CollectServers stats only
func (s *Stats) CollectServers(reload bool) {
	var online, indexable int
	s.data.EachServerInfo(func(_ string, server *model.MatrixServer) {
		if server.Online {
			online++
		}
		if server.Indexable {
			indexable++
		}
	})

	if err := s.data.SetIndexOnlineServers(online); err != nil {
		log.Println("cannot set online servers count", err)
	}

	if err := s.data.SetIndexIndexableServers(indexable); err != nil {
		log.Println("cannot set indexable servers count", err)
	}

	if err := s.data.SetIndexBlockedServers(s.block.Len()); err != nil {
		log.Println("cannot set blocked servers count", err)
	}

	if reload {
		s.reload()
	}
}

// Collect all stats from repository
func (s *Stats) Collect() {
	if s.collecting {
		log.Println("stats collection already in progress, ignoring request")
		return
	}
	s.collecting = true
	defer func() { s.collecting = false }()

	s.CollectServers(false)

	var rooms int
	s.data.EachRoom(func(_ string, _ *model.MatrixRoom) {
		rooms++
	})
	if err := s.data.SetIndexParsedRooms(rooms); err != nil {
		log.Println("cannot set parsed rooms count", err)
	}
	if err := s.data.SetIndexIndexedRooms(s.index.Len()); err != nil {
		log.Println("cannot set indexed rooms count", err)
	}
	banned, berr := s.data.GetBannedRooms()
	if berr != nil {
		log.Println("cannot get banned rooms count", berr)
	}
	if err := s.data.SetIndexBannedRooms(len(banned)); err != nil {
		log.Println("cannot set banned rooms count", err)
	}
	reported, rerr := s.data.GetReportedRooms()
	if rerr != nil {
		log.Println("cannot get reported rooms count", rerr)
	}
	if err := s.data.SetIndexReportedRooms(len(reported)); err != nil {
		log.Println("cannot set reported rooms count", err)
	}

	s.reload()
	s.sendWebhook()
}

// sendWebhook send request to webhook if provided
func (s *Stats) sendWebhook() {
	if s.webhook == "" {
		return
	}

	payload, err := json.Marshal(webhookPayload{
		Username: s.webhookUser,
		Markdown: s.getWebhookText(),
	})
	if err != nil {
		log.Printf("webhook payload marshaling failed: %v", err)
		return
	}

	req, err := http.NewRequest("POST", s.webhook, bytes.NewReader(payload))
	if err != nil {
		log.Printf("webhook request marshaling failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("webhook sending failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		log.Printf("backend returned HTTP %d: %s %v", resp.StatusCode, string(body), err)
	}
}

func (s *Stats) getWebhookText() string {
	var text strings.Builder
	text.WriteString("**stats have been collected**\n\n")

	serversDiff := s.stats.Servers.Online - s.prev.Servers.Online
	roomsDiff := s.stats.Rooms.Indexed - s.prev.Rooms.Indexed

	text.WriteString(fmt.Sprintf("* `%d` `%s%d` servers online (`%d` blocked)\n", s.stats.Servers.Online, getSymbol(serversDiff), abs(serversDiff), s.stats.Servers.Blocked))
	text.WriteString(fmt.Sprintf("* `%d` `%s%d` rooms (`%d` blocked, `%d` reported)\n", s.stats.Rooms.Indexed, getSymbol(roomsDiff), abs(roomsDiff), s.stats.Rooms.Banned, s.stats.Rooms.Reported))
	text.WriteString("\n---\n\n")

	discovery := s.stats.Discovery.FinishedAt.Sub(s.stats.Discovery.StartedAt)
	discoveryPrev := s.prev.Discovery.FinishedAt.Sub(s.prev.Discovery.StartedAt)
	discoveryDiff := discovery - discoveryPrev

	parsing := s.stats.Parsing.FinishedAt.Sub(s.stats.Parsing.StartedAt)
	parsingPrev := s.prev.Parsing.FinishedAt.Sub(s.prev.Parsing.FinishedAt)
	parsingDiff := parsing - parsingPrev

	indexing := s.stats.Indexing.FinishedAt.Sub(s.stats.Indexing.StartedAt)
	indexingPrev := s.prev.Indexing.FinishedAt.Sub(s.prev.Indexing.StartedAt)
	indexingDiff := indexing - indexingPrev

	total := discovery + parsing + indexing
	totalPrev := discoveryPrev + parsingPrev + indexingPrev
	totalDiff := total - totalPrev

	text.WriteString(fmt.Sprintf("* `%s` `%s%s` took discovery process\n", discovery.String(), getSymbol(discoveryDiff), discoveryDiff.String()))
	text.WriteString(fmt.Sprintf("* `%s` `%s%s` took parsing process\n", parsing.String(), getSymbol(parsingDiff), parsingDiff.String()))
	text.WriteString(fmt.Sprintf("* `%s` `%s%s` took indexing process\n", indexing.String(), getSymbol(indexingDiff), indexingDiff.String()))
	text.WriteString(fmt.Sprintf("* `%s` `%s%s` total\n", total.String(), getSymbol(totalDiff), totalDiff.String()))

	return text.String()
}

type Number interface {
	constraints.Float | constraints.Integer | constraints.Signed | constraints.Unsigned
}

// diffText tries to find difference and return it in a human-friendly way, e.g. prev=1, curr=5 -> "+4"
func getSymbol[T Number](diff T) string {
	if diff == 0 {
		return ""
	}

	if diff > 0 {
		return "+"
	}
	return "-"
}

func abs[T Number](number T) T {
	if number < 0 {
		return 0 - number
	}
	return number
}
