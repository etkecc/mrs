package kit

import "net"

// AnonymizeIP masks an IP for GDPR-friendly logging: IPv4 keeps /24, IPv6 keeps /48; strip any port first.
func AnonymizeIP(ip string) string {
	if ip == "" {
		return ""
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ip // not an ip
	}

	if v4 := parsedIP.To4(); v4 != nil {
		return v4.Mask(net.CIDRMask(24, 32)).String()
	}
	return parsedIP.Mask(net.CIDRMask(48, 128)).String()
}

// IsValidIP reports if ipStr is routable and public, excluding unspecified, loopback, private, multicast, link-local.
func IsValidIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	return !ip.IsUnspecified() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsMulticast() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}
