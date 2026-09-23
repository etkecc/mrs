package echobasicauth

import (
	"net"

	"github.com/labstack/echo/v4"
)

// ClientIP resolves the client address, trusting forwarded headers only when echo.IPExtractor is set
func ClientIP(c echo.Context) string {
	ip := getClientIP(c)
	if parsed := net.ParseIP(ip); parsed != nil {
		return parsed.String() // canonicalize
	}
	return "" // invalid
}

// getClientIP from echo's IP extractor or from remote address
func getClientIP(c echo.Context) string {
	if e := c.Echo(); e != nil && e.IPExtractor != nil {
		return c.RealIP()
	}

	host, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err != nil {
		return c.Request().RemoteAddr
	}
	return host
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
