# echo basic auth

Basic Auth middleware with constant time equality checks and optional IP whitelisting for Echo framework.
CIDRs are supported for IP whitelisting as well

## Usage

plaintext:

```go
auth := &echobasicauth.Auth{Login: "test", Password: "test", IPs: []string{"127.0.0.1", "10.0.0.0/24"}}
e.Use(echobasicauth.NewMiddleware(auth))
// or you can use echobasicauth.NewValidator(auth) if you want to define the middleware yourself
```

bcrypt:

```go
auth := &echobasicauth.Auth{Login: "test", Password: "$2a$10$XajjQvNhvvRt5GSeFk1xFeyqRrsxkhBkUiQeg0dt.wU1qD4aFDcga", IPs: []string{"127.0.0.1", "10.0.0.0/24"}}
```

(and bcrypted `Login` supported as well, just in case)

With several `Auth` values, a request passes when any one of them accepts the credentials; an entry
with an empty `IPs` list accepts its login from any IP.

`NewMiddleware()` with an empty auth list returns a middleware that rejects every request.

## IP whitelist

Plain IP entries are normalized before matching, so `::1` and `0:0:0:0:0:0:0:1` (and upper/lowercase hex)
are equivalent. CIDRs are supported as well.

By default, IP rules match the transport peer and client headers are ignored:

```go
// no e.IPExtractor set (the default)
e.Use(echobasicauth.NewMiddleware(auth))
```

Proxied deployments can opt in to checking the original client instead of the proxy address:

```go
e.IPExtractor = echo.ExtractIPFromXFFHeader()
```

**Security note:** this puts the IP gate on `X-Forwarded-For`. Echo's default extractor trusts
loopback, link-local, and private peers, so any client that reaches the app directly from a private
IP (office, VPN, same-host proxy) can set `X-Forwarded-For` to an allowlisted IP and pass the IP
check; with the bare extractor the IP gate provides no protection against such clients. When
combining the IP allowlist with an extractor, restrict the trust to your proxy range and disable
the broad defaults:

```go
proxyRange, _ := net.ParseCIDR("10.0.0.0/8") // your proxy range
e.IPExtractor = echo.ExtractIPFromXFFHeader(
	echo.TrustPrivateNet(false),
	echo.TrustLinkLocal(false),
	echo.TrustIPRange(proxyRange),
)
// keep the default TrustLoopback(true) only when your proxy sits on 127.0.0.1
```

`SetIPs` applies a new allowlist atomically for concurrent request processing; the exported `IPs`
field is for config-time population, and reassigning it from another goroutine is a data race.
Invalid entries never open the whitelist: an all-invalid list denies everybody, and `Validate`
reports the typos, so they can fail the boot instead of the check:

```go
if err := auth.Validate(); err != nil {
    return err
}
```

Failed attempts are logged at WARN with the anonymized client address; Echo's default level is ERROR, so
raise it to see them: `e.Logger.SetLevel(log.WARN)`.

### IP validation without credentials

```go
auth := &echobasicauth.Auth{IPs: []string{"127.0.0.1", "10.0.0.0/24"}}
if auth.AllowedIP(echobasicauth.ClientIP(c)) {
    // IP is allowed
}
```

Use `ClientIP` instead of `c.RealIP()`: it ignores client headers unless `e.IPExtractor` is set, and it
returns an empty string for unparseable addresses, which the allowlist denies.
