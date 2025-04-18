package services

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/etkecc/mrs/internal/utils"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
)

// Plausible - plausible analytics service
type Plausible struct {
	cfg ConfigService
}

func NewPlausible(cfg ConfigService) *Plausible {
	return &Plausible{cfg: cfg}
}

func (p *Plausible) Enabled() bool {
	if p.cfg.Get().Plausible == nil {
		return false
	}

	return p.cfg.Get().Plausible.Host != "" && p.cfg.Get().Plausible.Domain != ""
}

// TrackSearch - track search event
func (p *Plausible) TrackSearch(ctx context.Context, incomingReq *http.Request, ip, query string) {
	log := zerolog.Ctx(ctx)
	if !p.Enabled() {
		return
	}
	query = strings.TrimSpace(strings.ToLower(query))

	if query == "" {
		return
	}

	uri := url.URL{
		Scheme: "https",
		Host:   p.cfg.Get().Plausible.Host,
		Path:   "/api/event",
	}
	data := map[string]any{
		"name":     "Search",
		"url":      incomingReq.URL.String(),
		"domain":   p.cfg.Get().Plausible.Domain,
		"referrer": incomingReq.Referer(),
		"props": map[string]any{
			"query": query,
		},
	}
	datab, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal plausible event")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, utils.DefaultTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri.String(), bytes.NewReader(datab))
	if err != nil {
		log.Error().Err(err).Msg("cannot create plausible request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", incomingReq.UserAgent())
	req.Header.Set("X-Forwarded-For", ip)

	resp, err := utils.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("cannot send plausible request")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		log.Error().Int("status", resp.StatusCode).Msg("unexpected plausible response")
	}
}

// TrackOpen - track room open event - when user clicks on the room link
func (p *Plausible) TrackOpen(ctx context.Context, incomingReq *http.Request, ip, roomAlias string) {
	log := zerolog.Ctx(ctx)
	if !p.Enabled() {
		return
	}
	roomAlias = strings.TrimSpace(strings.ToLower(roomAlias))

	if roomAlias == "" {
		return
	}

	uri := url.URL{
		Scheme: "https",
		Host:   p.cfg.Get().Plausible.Host,
		Path:   "/api/event",
	}
	data := map[string]any{
		"name":     "Open",
		"url":      incomingReq.URL.String(),
		"domain":   p.cfg.Get().Plausible.Domain,
		"referrer": incomingReq.Referer(),
		"props": map[string]any{
			"room": roomAlias,
		},
	}
	log.Info().Any("data", data).Msg("plausible Open event")
	datab, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal plausible event")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, utils.DefaultTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri.String(), bytes.NewReader(datab))
	if err != nil {
		log.Error().Err(err).Msg("cannot create plausible request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", incomingReq.UserAgent())
	req.Header.Set("X-Forwarded-For", ip)

	resp, err := utils.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("cannot send plausible request")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		log.Error().Int("status", resp.StatusCode).Msg("unexpected plausible response")
	}
}
