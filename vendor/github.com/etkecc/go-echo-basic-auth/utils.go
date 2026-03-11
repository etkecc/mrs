package echobasicauth

import (
	"crypto/subtle"
	"net"
	"strings"
)

// equals performs equality check in constant time
func equals(str1, str2 string) bool {
	b1 := []byte(str1)
	b2 := []byte(str2)
	return subtle.ConstantTimeEq(int32(len(b1)), int32(len(b2))) == 1 && subtle.ConstantTimeCompare(b1, b2) == 1 //nolint:gosec // integer overflow is not expected
}

// anonymizeIP drops the last octet of the IPv4 and IPv6 address to anonymize it
func anonymizeIP(ip string) string {
	if ip == "" {
		return ""
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ip // not an ip
	}

	// IPv4
	if parsedIP.To4() != nil {
		ipParts := strings.Split(parsedIP.String(), ".")
		if len(ipParts) == 4 {
			ipParts[3] = "0"
			return strings.Join(ipParts, ".")
		}
	}

	// IPv6
	ipParts := strings.Split(parsedIP.String(), ":")
	if len(ipParts) > 0 {
		ipParts[len(ipParts)-1] = "0"
		return strings.Join(ipParts, ":")
	}
	return ip // not an ip
}
