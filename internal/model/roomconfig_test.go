package model

import (
	"testing"
)

func TestParseRoomConfig_BasicCases(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		wantRest string
		wantCfg  RoomConfig
	}{
		{
			name:     "Empty topic",
			topic:    "",
			wantRest: "",
			wantCfg:  RoomConfig{},
		},
		{
			name:     "No config",
			topic:    "General chat for everyone",
			wantRest: "General chat for everyone",
			wantCfg:  RoomConfig{},
		},
		{
			name:     "Config present with language & email",
			topic:    "Welcome (MRS-language:EN|email:test@example.com-MRS)",
			wantRest: "Welcome",
			wantCfg:  RoomConfig{Language: "EN", Email: "test@example.com"},
		},
		{
			name:     "Config with noindex true",
			topic:    "(MRS-noindex:true-MRS) Room for admins",
			wantRest: "Room for admins",
			wantCfg:  RoomConfig{Noindex: true},
		},
		{
			name:     "Config with noindex yes",
			topic:    "My Topic (MRS-noindex:yes-MRS)",
			wantRest: "My Topic",
			wantCfg:  RoomConfig{Noindex: true},
		},
		{
			name:     "Config with noindex 1",
			topic:    "Foo (MRS-noindex:1-MRS)",
			wantRest: "Foo",
			wantCfg:  RoomConfig{Noindex: true},
		},
		{
			name:     "Config with email only",
			topic:    "x (MRS-email:user@domain.org-MRS)",
			wantRest: "x",
			wantCfg:  RoomConfig{Email: "user@domain.org"},
		},
		{
			name:     "Config with language only, lower case",
			topic:    "(MRS-language:de-MRS)X",
			wantRest: "X",
			wantCfg:  RoomConfig{Language: "DE"},
		},
		{
			name:     "Config with language as garbage",
			topic:    "Hello (MRS-language:zzz-MRS)",
			wantRest: "Hello",
			wantCfg:  RoomConfig{},
		},
		{
			name:     "Config with bad email",
			topic:    "t (MRS-email:bademail-MRS)",
			wantRest: "t",
			wantCfg:  RoomConfig{},
		},
		{
			name:     "Trim spaces after removal",
			topic:    "   (MRS-noindex:yes-MRS)   ",
			wantRest: "",
			wantCfg:  RoomConfig{Noindex: true},
		},
		{
			name:     "Multiple config pairs, some bad/unknown",
			topic:    "Here (MRS-language:EN|foo:bar|email:abc@b.co|noindex:false|bla:blub-MRS)",
			wantRest: "Here",
			wantCfg:  RoomConfig{Language: "EN", Email: "abc@b.co"},
		},
		{
			name:     "Config at end, after normal text",
			topic:    "Discussion goes here (MRS-language:EN-MRS)",
			wantRest: "Discussion goes here",
			wantCfg:  RoomConfig{Language: "EN"},
		},
		{
			name:     "Config at start, then text",
			topic:    "(MRS-language:EN-MRS) Discussion",
			wantRest: "Discussion",
			wantCfg:  RoomConfig{Language: "EN"},
		},
		{
			name:     "Noindex false (should be false)",
			topic:    "abc(MRS-noindex:false-MRS)",
			wantRest: "abc",
			wantCfg:  RoomConfig{},
		},
		{
			name:     "Noindex unknown (should be false)",
			topic:    "abc(MRS-noindex:no-MRS)",
			wantRest: "abc",
			wantCfg:  RoomConfig{},
		},
		{
			name:     "Config with extra | at end",
			topic:    "abc(MRS-language:EN|email:x@y.z|-MRS)",
			wantRest: "abc",
			wantCfg:  RoomConfig{Language: "EN", Email: "x@y.z"},
		},
		{
			name:     "Duplicate keys, language last wins",
			topic:    "abc(MRS-language:DE|language:EN-MRS)",
			wantRest: "abc",
			wantCfg:  RoomConfig{Language: "EN"},
		},
		{
			name:     "Duplicate keys, email first wins",
			topic:    "abc(MRS-email:x@y.z|email:z@a.com-MRS)",
			wantRest: "abc",
			wantCfg:  RoomConfig{Email: "z@a.com"},
		},
		{
			name:     "Overlapping tags, only first applies",
			topic:    "A (MRS-language:EN-MRS) B (MRS-language:DE-MRS) C",
			wantRest: "A  B (MRS-language:DE-MRS) C",
			wantCfg:  RoomConfig{Language: "EN"},
		},
		{
			name:     "Odd case: opening tag but no end tag",
			topic:    "abc (MRS-language:EN|email:x@y.z)",
			wantRest: "abc (MRS-language:EN|email:x@y.z)",
			wantCfg:  RoomConfig{},
		},
		{
			name:     "Config with colons in value",
			topic:    `(MRS-language:EN|email:"foo:bar"@x.y-MRS) zzz`,
			wantRest: "zzz",
			wantCfg:  RoomConfig{Language: "EN", Email: `"foo:bar"@x.y`},
		},
		{
			name:     "Only start tag with no config",
			topic:    "(MRS--MRS) hey",
			wantRest: "hey",
			wantCfg:  RoomConfig{},
		},
		{
			name:     "Config values with extra spaces",
			topic:    "(MRS-language: en   | email:  test2@ex.com  -MRS) rest",
			wantRest: "rest",
			wantCfg:  RoomConfig{Language: "EN", Email: "test2@ex.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest, cfg := ParseRoomConfig(tt.topic)
			if rest != tt.wantRest {
				t.Errorf("rest: got [%s], want [%s]", rest, tt.wantRest)
			}
			if (cfg == nil && !tt.wantCfg.IsEmpty()) || (cfg != nil && cfg.Language != tt.wantCfg.Language) {
				t.Errorf("Language: got %q, want %q", cfg.Language, tt.wantCfg.Language)
			}
			if (cfg == nil && !tt.wantCfg.IsEmpty()) || (cfg != nil && cfg.Email != tt.wantCfg.Email) {
				t.Errorf("Email: got %q, want %q", cfg.Email, tt.wantCfg.Email)
			}
			wantNoindex := tt.wantCfg.Noindex
			haveNoindex := false
			if cfg != nil {
				haveNoindex = cfg.Noindex
			}
			if haveNoindex != wantNoindex {
				t.Errorf("Noindex: got %v, want %v", haveNoindex, wantNoindex)
			}
		})
	}
}

func TestRoomConfig_IsEmpty(t *testing.T) {
	empty := &RoomConfig{}
	if !empty.IsEmpty() {
		t.Error("Expected empty RoomConfig to be empty")
	}
	if (&RoomConfig{Language: "EN"}).IsEmpty() {
		t.Error("Should not be empty with language")
	}
	if (&RoomConfig{Email: "a@b.com"}).IsEmpty() {
		t.Error("Should not be empty with email")
	}
	if (&RoomConfig{Noindex: true}).IsEmpty() {
		t.Error("Should not be empty with noindex true")
	}
	if ((*RoomConfig)(nil)).IsEmpty() != true {
		t.Error("Nil pointer must be empty")
	}
}
