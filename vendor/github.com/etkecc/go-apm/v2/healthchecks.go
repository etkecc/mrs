package apm

import "io"

// Healthchecks mirrors gitlab.com/etke.cc/go/healthchecks so projects that don't need it avoid the direct dependency.
type Healthchecks interface {
	Fail(optionalBody ...io.Reader)
}

// HealthcheckFail sends a healthcheck fail event (if healthchecks are configured)
func HealthcheckFail(optionalBody ...io.Reader) {
	if hc != nil {
		hc.Fail(optionalBody...)
	}
}
