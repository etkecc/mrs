package httpclient

import (
	"fmt"
	"net"
	"syscall"
)

// disallowedNets are ranges stdlib net.IP.Is* misses: v4-in-v6 embedders, CGNAT, 0.0.0.0/8, cloud-metadata.
var disallowedNets = []*net.IPNet{
	mustCIDR("64:ff9b::/96"),       // NAT64
	mustCIDR("64:ff9b:1::/48"),     // NAT64 local-use
	mustCIDR("2002::/16"),          // 6to4
	mustCIDR("2001::/32"),          // Teredo
	mustCIDR("::/96"),              // IPv4-compatible (deprecated); ::ffff: mapped form is handled by To4
	mustCIDR("100.64.0.0/10"),      // CGNAT
	mustCIDR("0.0.0.0/8"),          // "this network"
	mustCIDR("100.100.100.200/32"), // Alibaba metadata
}

// mustCIDR parses a compile-time-fixed CIDR; a bad literal is an impossible-startup bug, so panic.
func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

// isDisallowedIP checks loopback/private/link-local/unspecified/multicast/disallowedNets; To4() avoids a v6 bypass.
func isDisallowedIP(ip net.IP) bool {
	norm := ip
	if v4 := ip.To4(); v4 != nil {
		norm = v4
	}
	if norm.IsLoopback() ||
		norm.IsPrivate() ||
		norm.IsLinkLocalUnicast() ||
		norm.IsUnspecified() ||
		norm.IsMulticast() {
		return true
	}
	for _, n := range disallowedNets {
		if n.Contains(norm) {
			return true
		}
	}
	return false
}

// dialGuard is Dialer.Control from WithDialGuard: checks the resolved/pinned IP at dial time, the last honest moment.
func dialGuard(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("dial guard: malformed address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || isDisallowedIP(ip) {
		return fmt.Errorf("dial guard: refusing to dial disallowed address %s", host)
	}
	return nil
}
