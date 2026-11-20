package echobasicauth

import (
	"net"

	"github.com/labstack/echo/v5"
)

// ClientIP resolves the client address, trusting forwarded headers only when echo.IPExtractor is set
func ClientIP(c *echo.Context) string {
	if parsed := net.ParseIP(c.RealIP()); parsed != nil {
		return parsed.String() // canonicalize
	}
	return "" // invalid
}

// anonymizeIP drops the last octet of the IPv4 and IPv6 address to anonymize it
func anonymizeIP(ip string) string {
	if ip == "" {
		return ""
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "invalid" // not an ip
	}

	if v4 := parsedIP.To4(); v4 != nil {
		return v4.Mask(net.CIDRMask(24, 32)).String()
	}
	return parsedIP.Mask(net.CIDRMask(48, 128)).String()
}
