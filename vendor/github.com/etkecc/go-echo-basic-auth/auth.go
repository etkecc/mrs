package echobasicauth

import (
	"crypto/subtle"
	"net"
	"slices"
	"strings"

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
		sanitizedPath := strings.ReplaceAll(c.Request().URL.Path, "\n", "")
		sanitizedPath = strings.ReplaceAll(sanitizedPath, "\r", "")
		for idx, auth := range auths {
			allowedIP := isIPAllowed(validIPs[idx], validCIDRs[idx], c.RealIP())
			match := Equals(auth.Login, login) && Equals(auth.Password, password)
			if match && allowedIP {
				c.Set(ContextLoginKey, login)
				c.Logger().Infof("authorization attempt from %s to %s (allowed_ip==%t and allowed_credentials==%t)", c.RealIP(), sanitizedPath, allowedIP, match)
				return true, nil
			}
		}
		c.Logger().Infof("authorization attempt from %s to %s (allowed_ip==%t or allowed_credentials==%t)", c.RealIP(), sanitizedPath, false, false)

		return false, nil
	}
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
