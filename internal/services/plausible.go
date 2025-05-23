package services

import (
	"bytes"
	"context"
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

	uri := url.URL{
		Scheme: "https",
		Host:   p.cfg.Get().Plausible.Host,
		Path:   "/api/event",
	}
	data := map[string]any{
		"name":     evt.Name,
		"url":      evt.URL,
		"domain":   p.cfg.Get().Plausible.Domain,
		"referrer": evt.Referrer,
		"props":    evt.Props,
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
	req.Header.Set("User-Agent", evt.UserAgent)
	req.Header.Set("X-Forwarded-For", evt.XForwardedFor)

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
