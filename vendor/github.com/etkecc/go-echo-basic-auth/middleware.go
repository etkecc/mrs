package echobasicauth

import (
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ContextLoginKey is the key used to store the login after successful auth in the context
const ContextLoginKey = "echo-basic-auth.login"

// NewValidator returns a new BasicAuthValidator
func NewValidator(auths ...*Auth) middleware.BasicAuthValidator {
	auths = slices.DeleteFunc(auths, func(a *Auth) bool { return a == nil })
	if len(auths) == 0 {
		return nil
	}
	return func(login, password string, c echo.Context) (bool, error) {
		var wasIPAllowed, wasAuthAllowed bool
		for _, auth := range auths {
			allowedIP := auth.AllowedIP(c.RealIP())
			if allowedIP {
				wasIPAllowed = true
			}
			match := equals(auth.Login, login) && equals(auth.Password, password)
			if match {
				wasAuthAllowed = true
			}

			if match && allowedIP {
				c.Set(ContextLoginKey, login)
				return true, nil
			}
		}

		logAttempt(c, wasIPAllowed, wasAuthAllowed)
		return false, nil
	}
}

// logAttempt logs a failed authentication attempt
func logAttempt(c echo.Context, wasIPAllowed, wasAuthAllowed bool) {
	requestPath := strings.ReplaceAll(strings.ReplaceAll(c.Request().URL.Path, "\n", ""), "\r", "")
	c.Logger().Warnf(
		`%s - FAIL [%s] "%s %s %s" 401 0 "-" "Auth: false (ip: %t; creds: %t)"`,
		anonymizeIP(c.RealIP()),
		time.Now().Format("2/Jan/2006:15:04:05 -0700"),
		c.Request().Method,
		requestPath,
		c.Request().Proto,
		wasIPAllowed,
		wasAuthAllowed,
	)
}

// NewMiddleware returns a new BasicAuth middleware instance
func NewMiddleware(auths ...*Auth) echo.MiddlewareFunc {
	return middleware.BasicAuth(NewValidator(auths...))
}
