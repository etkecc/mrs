package services

import (
	"testing"

	"github.com/etkecc/mrs/internal/model"
)

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
