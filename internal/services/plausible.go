package services

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
	"github.com/goccy/go-json"
)

type PlausibleService interface {
	Track(ctx context.Context, evt *model.AnalyticsEvent)
}

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

func (p *Plausible) Track(ctx context.Context, evt *model.AnalyticsEvent) {
	log := apm.Log(ctx)
	if !p.Enabled() {
		return
	}

	eventURL := p.eventURL(evt.URL)
	ctx, cancel := context.WithTimeout(ctx, utils.DefaultTimeout)
	defer cancel()
	req, err := p.buildRequest(ctx, evt, eventURL)
	if err != nil {
		log.Error().Err(err).Msg("cannot build plausible request")
		return
	}

	// the wire on demand: flip to debug to see what left MRS. ip + user_agent are the visitor-hash terms.
	log.Debug().
		Str("name", evt.Name).
		Str("referrer", evt.Referrer).
		Str("user_agent", evt.UserAgent).
		Str("ip", evt.ClientIP).
		Msg("plausible event payload")

	resp, err := utils.Do(req)
	if err != nil {
		log.Warn().Err(err).Msg("cannot send plausible request")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		log.Warn().Int("status", resp.StatusCode).Msg("unexpected plausible response")
	}
	// 202 comes back even on a silent drop; this header is the only tell. resp is non-nil past the err guard.
	if resp.Header.Get("x-plausible-dropped") == "1" {
		log.Warn().Str("url", eventURL).Msg("plausible silently dropped the event")
	}
}

// buildRequest assembles the event POST: the body, and every header Plausible's visitor hash reads.
func (p *Plausible) buildRequest(ctx context.Context, evt *model.AnalyticsEvent, eventURL string) (*http.Request, error) {
	data := map[string]any{
		"name":     evt.Name,
		"url":      eventURL,
		"domain":   p.cfg.Get().Plausible.Domain,
		"referrer": evt.Referrer,
		"props":    evt.Props,
	}
	datab, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal plausible event: %w", err)
	}

	uri := url.URL{
		Scheme: "https",
		Host:   p.cfg.Get().Plausible.Host,
		Path:   "/api/event",
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri.String(), bytes.NewReader(datab))
	if err != nil {
		return nil, fmt.Errorf("cannot create plausible request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", evt.UserAgent)
	if evt.ClientIP != "" {
		// Traefik rewrites X-Forwarded-* on the hop and leaves custom headers alone; Plausible reads all three last-wins.
		req.Header.Set("CF-Connecting-IP", evt.ClientIP)
		req.Header.Set("X-Forwarded-For", evt.ClientIP)
		req.Header.Set("X-Plausible-IP", evt.ClientIP)
	}

	return req, nil
}

// eventURL resolves the bare request path against Public.API so the page lands under our domain, not Plausible's hostless "(none)" bucket.
func (p *Plausible) eventURL(raw string) string {
	pub := p.cfg.Get().Public
	if pub == nil || pub.API == "" {
		return raw
	}
	base, err := url.Parse(pub.API)
	if err != nil {
		return raw
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	// resolve only a genuine relative path; an authority-bearing ref (//host) would let a foreign host into the dashboard.
	if ref.IsAbs() || ref.Host != "" {
		return raw
	}
	return base.ResolveReference(ref).String()
}
