package echobasicauth

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// Auth model
type Auth struct {
	Login    string   `json:"login" yaml:"login"` // Basic auth login
	Password string   `json:"password" yaml:"password"`
	IPs      []string `json:"ips" yaml:"ips"` // Allowed IPs and CIDRs; use SetIPs for runtime changes

	mu          sync.Mutex   // guards the parsed rules below
	parsed      bool         // whether parsedIPs and parsedCIDRs match the current IPs field
	parsedFrom  []string     // copy of the IPs field the parsed rules were built from
	parsedIPs   []string     // parsed plain IPs from the IPs field, used by AllowedIP
	parsedCIDRs []*net.IPNet // parsed CIDRs from the IPs field, used by AllowedIP
}

// parseEntry classifies an allowlist entry as a CIDR or a plain IP, returning both as nil when it is neither
func parseEntry(entry string) (*net.IPNet, net.IP) {
	if _, ipnet, err := net.ParseCIDR(entry); err == nil {
		return ipnet, nil
	}
	return nil, net.ParseIP(entry)
}

// rules returns the parsed allowlist, rebuilt whenever the IPs field changed since the last call
func (a *Auth) rules() ([]string, []*net.IPNet) {
	a.mu.Lock()
	ips := slices.Clone(a.IPs)
	current := a.parsed && slices.Equal(a.parsedFrom, ips)
	parsedIPs, parsedCIDRs := a.parsedIPs, a.parsedCIDRs
	if !current {
		parsedIPs = []string{}
		parsedCIDRs = []*net.IPNet{}
		for _, entry := range ips {
			ipnet, ip := parseEntry(entry)
			if ipnet != nil {
				parsedCIDRs = append(parsedCIDRs, ipnet)
			} else if ip != nil {
				parsedIPs = append(parsedIPs, ip.String())
			}
		}
		a.parsed = true
		a.parsedFrom = ips
		a.parsedIPs = parsedIPs
		a.parsedCIDRs = parsedCIDRs
	}
	a.mu.Unlock()
	return parsedIPs, parsedCIDRs
}

// SetIPs atomically replaces the IP allowlist, safe for concurrent request processing
func (a *Auth) SetIPs(ips []string) {
	a.mu.Lock()
	a.IPs = slices.Clone(ips)
	a.mu.Unlock()
}

// compare hash/expected with raw/input, both bcrypted and plaintext.
func (a *Auth) compare(hash, raw string) bool {
	hashb := []byte(hash)
	rawb := []byte(raw)
	if _, err := bcrypt.Cost(hashb); err != nil {
		return subtle.ConstantTimeCompare(hashb, rawb) == 1
	}
	return bcrypt.CompareHashAndPassword(hashb, rawb) == nil
}

// AllowedIP checks if the given IP is allowed by this Auth's IP rules
func (a *Auth) AllowedIP(ip string) bool {
	parsedIPs, parsedCIDRs := a.rules()
	if len(parsedIPs) == 0 && len(parsedCIDRs) == 0 {
		// No configured entries mean no IP restriction, unparseable entries deny everyone
		return len(a.IPs) == 0
	}

	if parsed := net.ParseIP(ip); parsed != nil {
		ip = parsed.String() // canonicalize peer so the plain-IP match is IP-equality
	}
	if len(parsedIPs) != 0 && slices.Contains(parsedIPs, ip) {
		return true
	}
	if len(parsedCIDRs) != 0 {
		for _, ipnet := range parsedCIDRs {
			if ipnet.Contains(net.ParseIP(ip)) {
				return true
			}
		}
	}

	return false
}

// Match checks if the given login and password match.
func (a *Auth) Match(login, password string) bool {
	notEmpty := login != "" && password != ""
	return notEmpty && a.compare(a.Login, login) && a.compare(a.Password, password)
}

// Validate reports allowlist entries that are neither an IP nor a CIDR, so a typo fails the boot
func (a *Auth) Validate() error {
	errs := make([]error, 0, len(a.IPs))
	for _, entry := range a.IPs {
		ipnet, ip := parseEntry(entry)
		if ipnet == nil && ip == nil {
			errs = append(errs, fmt.Errorf("invalid IP or CIDR %q", entry))
		}
	}
	return errors.Join(errs...)
}
