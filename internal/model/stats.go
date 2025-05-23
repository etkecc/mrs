package model

import (
	"context"
	"net/http"
	"time"

	"github.com/etkecc/mrs/internal/model/mcontext"
)

// IndexStats structure
type IndexStats struct {
	Servers   IndexStatsServers `json:"servers"`
	Rooms     IndexStatsRooms   `json:"rooms"`
	Discovery IndexStatsTime    `json:"discovery"`
	Parsing   IndexStatsTime    `json:"parsing"`
	Indexing  IndexStatsTime    `json:"indexing"`
}

// IndexStatsServers structure
type IndexStatsServers struct {
	Online    int `json:"online"`
	Indexable int `json:"indexable"`
	Blocked   int `json:"blocked"`
}

// IndexStatsRooms structure
type IndexStatsRooms struct {
	Indexed  int `json:"indexed"`
	Parsed   int `json:"parsed"`
	Banned   int `json:"banned"`
	Reported int `json:"reported"`
}

// IndexStatsTime structure
type IndexStatsTime struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

// AnalyticsEvent structure
type AnalyticsEvent struct {
	Name          string            // Event name
	URL           string            // URL of the page where the event occurred
	Props         map[string]string // Event properties
	Referrer      string            // Referrer URL
	UserAgent     string            // User-Agent header of the incoming request
	XForwardedFor string            // X-Forwarded-For header of the incoming request
}

// NewAnalyticsEvent creates a new AnalyticsEvent
func NewAnalyticsEvent(cxt context.Context, name string, props map[string]string, req *http.Request) *AnalyticsEvent {
	evt := &AnalyticsEvent{
		Name:          name,
		URL:           req.URL.String(),
		Props:         props,
		Referrer:      req.Referer(),
		UserAgent:     req.UserAgent(),
		XForwardedFor: req.Header.Get("X-Forwarded-For"),
	}

	if evt.Referrer == "" {
		origin := mcontext.GetOrigin(cxt)
		if origin != "" {
			evt.Referrer = "https://" + origin + "/"
		}
	}
	if evt.UserAgent == "" {
		evt.UserAgent = "Synapse" // workaround
	}
	if evt.XForwardedFor == "" {
		evt.XForwardedFor = mcontext.GetIP(cxt)
	}

	return evt
}
