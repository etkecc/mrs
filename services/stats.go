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

	"gitlab.com/etke.cc/mrs/api/model"
)

type StatsRepository interface {
	DataRepository
	GetIndexStats() *model.IndexStats
	SetIndexOnlineServers(servers int) error
	SetIndexBlockedServers(servers int) error
	SetIndexRooms(rooms int) error
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
	stats       *model.IndexStats
	webhookUser string
	webhook     string
	collecting  bool
}

// NewStats service
func NewStats(data StatsRepository, blocklist Lenable, uiurl, webhook string) *Stats {
	if uiurl != "" {
		parsedUIURL, err := url.Parse(uiurl)
		if err == nil {
			uiurl = parsedUIURL.Hostname()
		}
	}
	stats := &Stats{data: data, block: blocklist, webhook: webhook, webhookUser: uiurl}
	stats.Reload()

	return stats
}

// Reload saved stats. Useful when you need to get updated timestamps, but don't want to parse whole db
func (s *Stats) Reload() {
	s.stats = s.data.GetIndexStats()
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

// Collect stats from repository
func (s *Stats) Collect() {
	if s.collecting {
		log.Println("stats collection already in progress, ignoring request")
		return
	}
	s.collecting = true
	defer func() { s.collecting = false }()

	if err := s.data.SetIndexOnlineServers(len(s.data.AllServers())); err != nil {
		log.Println("cannot set indexed servers count", err)
	}

	if err := s.data.SetIndexBlockedServers(s.block.Len()); err != nil {
		log.Println("cannot set blocked servers count", err)
	}

	var rooms int
	s.data.EachRoom(func(_ string, _ *model.MatrixRoom) {
		rooms++
	})
	if err := s.data.SetIndexRooms(rooms); err != nil {
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

	s.Reload()
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
	text.WriteString("**Stats have been collected**\n\n")

	text.WriteString(fmt.Sprintf("* `%d` servers online (`%d` blocked)\n", s.stats.Servers.Online, s.stats.Servers.Blocked))
	text.WriteString(fmt.Sprintf("* `%d` rooms (`%d` blocked, `%d` reported)\n", s.stats.Rooms.All, s.stats.Rooms.Banned, s.stats.Rooms.Reported))
	text.WriteString("\n---\n\n")

	discovery := s.stats.Discovery.FinishedAt.Sub(s.stats.Discovery.StartedAt)
	parsing := s.stats.Parsing.FinishedAt.Sub(s.stats.Parsing.StartedAt)
	indexing := s.stats.Indexing.FinishedAt.Sub(s.stats.Indexing.StartedAt)
	text.WriteString(fmt.Sprintf("* `%s` took discovery process\n", discovery.String()))
	text.WriteString(fmt.Sprintf("* `%s` took parsing process\n", parsing.String()))
	text.WriteString(fmt.Sprintf("* `%s` took indexing process\n", indexing.String()))
	text.WriteString(fmt.Sprintf("* `%s` total\n", (discovery + parsing + indexing).String()))

	return text.String()
}
