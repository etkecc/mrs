package controllers

import "testing"

func TestIsSafeImage(t *testing.T) {
	cases := []struct {
		contentType string
		want        bool
	}{
		{"image/png", true},
		{"image/jpeg; charset=binary", true},
		{"image/webp", true},
		{"image/svg+xml", false},
		{"text/html; charset=utf-8", false},
		{"", false},
		{"not a media type", false},
	}
	for _, tc := range cases {
		if got := isSafeImage(tc.contentType); got != tc.want {
			t.Errorf("isSafeImage(%q) = %v, want %v", tc.contentType, got, tc.want)
		}
	}
}
