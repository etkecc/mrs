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

// ContextLoginKey is the key used to store the login after successful auth in the context
const ContextLoginKey = "echo-basic-auth.login"

// NewValidator returns a new BasicAuthValidator
func NewValidator(auths ...*Auth) middleware.BasicAuthValidator {
	if len(auths) == 0 || auths[0] == nil {
		return nil
	}
	return func(login, password string, c echo.Context) (bool, error) {
		for _, auth := range auths {
			allowedIP := true
			if len(auth.IPs) != 0 {
				allowedIP = slices.Contains(auth.IPs, c.RealIP())
			}
			match := Equals(auth.Login, login) && Equals(auth.Password, password)
			if match && allowedIP {
				c.Set(ContextLoginKey, login)
				c.Logger().Infof("authorization attempt from %s to %s (allowed_ip=%t allowed_credentials=%t)", c.RealIP(), c.Request().URL.Path, allowedIP, match)
				return true, nil
			}
		}
		c.Logger().Infof("authorization attempt from %s to %s (allowed_ip=%t allowed_credentials=%t)", c.RealIP(), c.Request().URL.Path, false, false)

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
	return subtle.ConstantTimeEq(int32(len(b1)), int32(len(b2))) == 1 && subtle.ConstantTimeCompare(b1, b2) == 1
}
