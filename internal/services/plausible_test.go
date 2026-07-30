package services

import (
	"testing"

	"github.com/etkecc/mrs/internal/model"
)

func TestPlausibleBuildRequest(t *testing.T) {
	const clientIP = "192.0.2.1"
	ipHeaders := []string{"CF-Connecting-IP", "X-Forwarded-For", "X-Plausible-IP"}
	cases := []struct {
		name string
		ip   string
		want string
	}{
		{
			name: "a resolved IP lands on every header plausible might read",
			ip:   clientIP,
			want: clientIP,
		},
		{
			name: "an empty IP leaves all three alone",
			ip:   "",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgMock := NewMockConfigService(t)
			cfgMock.EXPECT().Get().Return(&model.Config{
				Plausible: &model.ConfigPlausible{Host: "plausible.example", Domain: "matrixrooms.info"},
			}).Maybe()
			p := NewPlausible(cfgMock)
			evt := &model.AnalyticsEvent{Name: "pageview", UserAgent: "Synapse", ClientIP: tc.ip}

			req, err := p.buildRequest(t.Context(), evt, "https://matrixrooms.info/room/x")
			if err != nil {
				t.Fatal(err)
			}

			for _, header := range ipHeaders {
				if got := req.Header.Get(header); got != tc.want {
					t.Errorf("%s = %q, want %q", header, got, tc.want)
				}
			}
			if got := req.Header.Get("User-Agent"); got != evt.UserAgent {
				t.Errorf("User-Agent = %q, want %q", got, evt.UserAgent)
			}
			if got := req.URL.String(); got != "https://plausible.example/api/event" {
				t.Errorf("URL = %q, want the plausible event endpoint", got)
			}
		})
	}
}

func TestPlausibleEventURL(t *testing.T) {
	const path = "/_matrix/federation/v1/publicRooms?q=matrix"
	cases := []struct {
		name string
		pub  *model.ConfigPublic
		raw  string
		want string
	}{
		{
			name: "resolves a bare path under the public host",
			pub:  &model.ConfigPublic{API: "https://matrixrooms.info"},
			raw:  path,
			want: "https://matrixrooms.info" + path,
		},
		{
			name: "nil public config falls back to the raw path",
			pub:  nil,
			raw:  path,
			want: path,
		},
		{
			name: "empty API falls back to the raw path",
			pub:  &model.ConfigPublic{API: ""},
			raw:  path,
			want: path,
		},
		{
			name: "authority-bearing ref is rejected, not resolved to a foreign host",
			pub:  &model.ConfigPublic{API: "https://matrixrooms.info"},
			raw:  "//evil.example/inject",
			want: "//evil.example/inject",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgMock := NewMockConfigService(t)
			cfgMock.EXPECT().Get().Return(&model.Config{Public: tc.pub}).Maybe()
			p := NewPlausible(cfgMock)
			if got := p.eventURL(tc.raw); got != tc.want {
				t.Errorf("eventURL(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
