package echobasicauth

import (
	"crypto/subtle"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Auth model
type Auth struct {
	Login    string   `json:"login" yaml:"login"`       // Basic auth login
	Password string   `json:"password" yaml:"password"` // Basic auth password
	IPs      []string `json:"ips" yaml:"ips"`           // Allowed IPs and CIDRs
}

// ContextLoginKey is the key used to store the login after successful auth in the context
const ContextLoginKey = "echo-basic-auth.login"

// NewValidator returns a new BasicAuthValidator
func NewValidator(auths ...*Auth) middleware.BasicAuthValidator {
	if len(auths) == 0 || auths[0] == nil {
		return nil
	}
	validIPs, validCIDRs := parseIPs(auths...)

	return func(login, password string, c echo.Context) (bool, error) {
		var wasIPAllowed, wasAuthAllowed bool
		for idx, auth := range auths {
			allowedIP := isIPAllowed(validIPs[idx], validCIDRs[idx], c.RealIP())
			if allowedIP {
				wasIPAllowed = true
			}
			match := Equals(auth.Login, login) && Equals(auth.Password, password)
			if match {
				wasAuthAllowed = true
			}

			if match && allowedIP {
				c.Set(ContextLoginKey, login)
				logAttempt(c, allowedIP, match, true)
				return true, nil
			}
		}

		logAttempt(c, wasIPAllowed, wasAuthAllowed, false)
		return false, nil
	}
}

// logAttempt logs the authentication attempt
func logAttempt(c echo.Context, wasIPAllowed, wasAuthAllowed, success bool) {
	user := "OK"
	status := "200"
	logfunc := c.Logger().Infof
	requestPath := strings.ReplaceAll(strings.ReplaceAll(c.Request().URL.Path, "\n", ""), "\r", "")
	if !success {
		user = "FAIL"
		status = "401"
		logfunc = c.Logger().Warnf
	}

	logfunc(
		`%s - %s [%s] "%s %s %s" %s 0 "-" "Auth: %t (ip: %t; creds: %t)"`,
		anonymizeIP(c.RealIP()),
		user,
		time.Now().Format("2/Jan/2006:15:04:05 -0700"),
		c.Request().Method,
		requestPath,
		c.Request().Proto,
		status,
		success,
		wasIPAllowed,
		wasAuthAllowed,
	)
}

// NewMiddleware returns a new BasicAuth middleware instance
func NewMiddleware(auths ...*Auth) echo.MiddlewareFunc {
	return middleware.BasicAuth(NewValidator(auths...))
}

// Equals performs equality check in constant time
func Equals(str1, str2 string) bool {
	b1 := []byte(str1)
	b2 := []byte(str2)
	return subtle.ConstantTimeEq(int32(len(b1)), int32(len(b2))) == 1 && subtle.ConstantTimeCompare(b1, b2) == 1 //nolint:gosec // integer overflow is not expected
}

// parseIPs parses the IPs and CIDRs
func parseIPs(auths ...*Auth) (validIPs map[int][]string, validCIDRs map[int][]*net.IPNet) {
	validIPs = map[int][]string{}
	validCIDRs = map[int][]*net.IPNet{}

	for idx, auth := range auths {
		validIPs[idx] = []string{}
		validCIDRs[idx] = []*net.IPNet{}
		for _, ip := range auth.IPs {
			if _, ipnet, err := net.ParseCIDR(ip); err == nil {
				validCIDRs[idx] = append(validCIDRs[idx], ipnet)
			} else {
				validIPs[idx] = append(validIPs[idx], ip)
			}
		}
	}
	return validIPs, validCIDRs
}

// isIPAllowed checks if the IP is allowed
func isIPAllowed(validIPs []string, validCIDRs []*net.IPNet, ip string) bool {
	allowedIP := true
	if len(validIPs) != 0 {
		allowedIP = slices.Contains(validIPs, ip)
	}
	if len(validCIDRs) != 0 {
		ip := net.ParseIP(ip)
		for _, ipnet := range validCIDRs {
			if ipnet.Contains(ip) {
				allowedIP = true
				break
			}
		}
	}

	return allowedIP
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
