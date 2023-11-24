package echobasicauth

import (
	"crypto/subtle"
	"slices"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Auth model
type Auth struct {
	Login    string   `json:"login" yaml:"login"`       // Basic auth login
	Password string   `json:"password" yaml:"password"` // Basic auth password
	IPs      []string `json:"ips" yaml:"ips"`           // Allowed IPs
}

// NewValidator returns a new BasicAuthValidator
func NewValidator(auth *Auth) middleware.BasicAuthValidator {
	if auth == nil {
		return nil
	}
	return func(login, password string, c echo.Context) (bool, error) {
		allowedIP := true
		if len(auth.IPs) != 0 {
			allowedIP = slices.Contains(auth.IPs, c.RealIP())
		}
		match := Equals(auth.Login, login) && Equals(auth.Password, password)
		c.Logger().Infof("authorization attempt from %s to %s (allowed_ip=%t allowed_credentials=%t)", c.RealIP(), c.Request().URL.Path, allowedIP, match)

		return match && allowedIP, nil
	}
}

// NewMiddleware returns a new BasicAuth middleware instance
func NewMiddleware(auth *Auth) echo.MiddlewareFunc {
	return middleware.BasicAuth(NewValidator(auth))
}

// Equals performs equality check in constant time
func Equals(str1, str2 string) bool {
	b1 := []byte(str1)
	b2 := []byte(str2)
	return subtle.ConstantTimeEq(int32(len(b1)), int32(len(b2))) == 1 && subtle.ConstantTimeCompare(b1, b2) == 1
}
