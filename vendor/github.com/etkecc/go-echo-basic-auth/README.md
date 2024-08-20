# echo basic auth

Basic Auth middleware with constant time equality checks and optional IP whitelisting for Echo framework

## Usage

```go
auth := &echobasicauth.Auth{Login: "test", Password: "test", IPs: []string{"127.0.0.1"}}
e.Use(echobasicauth.NewMiddleware(auth))
// or you can use echobasicauth.NewValidator(auth) if you want to define the middleware yourself
```
