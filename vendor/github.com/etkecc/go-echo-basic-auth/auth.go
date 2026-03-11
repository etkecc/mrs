package echobasicauth

import (
	"net"
	"slices"
	"sync"
)

// Auth model
type Auth struct {
	Login    string   `json:"login" yaml:"login"`       // Basic auth login
	Password string   `json:"password" yaml:"password"` //nolint:gosec // this is an auth model, the field is intentional
	IPs      []string `json:"ips" yaml:"ips"`           // Allowed IPs and CIDRs

	parsedIPs   []string     // lazily parsed plain IPs from the IPs field, used by AllowedIP
	parsedCIDRs []*net.IPNet // lazily parsed CIDRs from the IPs field, used by AllowedIP
	parseOnce   sync.Once    // ensures IPs are parsed only once, even under concurrent access
}

// parseIPs parses the IPs and CIDRs from the IPs field into internal state (called once lazily)
func (a *Auth) parseIPs() {
	a.parseOnce.Do(func() {
		a.parsedIPs = []string{}
		a.parsedCIDRs = []*net.IPNet{}
		for _, ip := range a.IPs {
			if _, ipnet, err := net.ParseCIDR(ip); err == nil {
				a.parsedCIDRs = append(a.parsedCIDRs, ipnet)
			} else if net.ParseIP(ip) != nil {
				a.parsedIPs = append(a.parsedIPs, ip)
			}
		}
	})
}

// AllowedIP checks if the given IP is allowed by this Auth's IP rules
func (a *Auth) AllowedIP(ip string) bool {
	a.parseIPs()
	if len(a.parsedIPs) == 0 && len(a.parsedCIDRs) == 0 {
		return true
	}

	if len(a.parsedIPs) != 0 && slices.Contains(a.parsedIPs, ip) {
		return true
	}

	if len(a.parsedCIDRs) != 0 {
		parsed := net.ParseIP(ip)
		for _, ipnet := range a.parsedCIDRs {
			if ipnet.Contains(parsed) {
				return true
			}
		}
	}

	return false
}
