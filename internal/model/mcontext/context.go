package mcontext

import "context"

type contextKey string

const (
	ipKey     contextKey = "ip"
	originKey contextKey = "origin"
)

// WithIP sets the IP address in the context
func WithIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ipKey, ip)
}

// GetIP retrieves the IP address from the context
func GetIP(ctx context.Context) string {
	val, ok := ctx.Value(ipKey).(string)
	if ok {
		return val
	}
	return ""
}

// WithOrigin sets the origin in the context
func WithOrigin(ctx context.Context, origin string) context.Context {
	return context.WithValue(ctx, originKey, origin)
}

// GetOrigin retrieves the origin from the context
func GetOrigin(ctx context.Context) string {
	val, ok := ctx.Value(originKey).(string)
	if ok {
		return val
	}
	return ""
}
