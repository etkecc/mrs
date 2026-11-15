package services

import (
	"testing"

	"github.com/etkecc/mrs/internal/model"
)

func TestSendModReportWithoutConfiguredClient(t *testing.T) {
	cfgMock := NewMockConfigService(t)
	cfgMock.EXPECT().Get().Return(&model.Config{
		Email: &model.ConfigEmail{Moderation: "mod@example.org"},
	}).Maybe()

	e := NewEmail(cfgMock)
	if err := e.SendModReport("body", "mod@example.org"); err == nil {
		t.Fatal("expected an unconfigured email error")
	}
}
