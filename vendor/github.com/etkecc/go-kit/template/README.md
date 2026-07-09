# template

[![Go Reference](https://pkg.go.dev/badge/github.com/etkecc/go-kit.svg)](https://pkg.go.dev/github.com/etkecc/go-kit/template)

Parse a template string, execute it with your data, get a string back. Three functions for three tempers: return the error, swallow it and fall back, or panic. That's the whole package.

Root module, zero dependencies.

```go
go get github.com/etkecc/go-kit
```

```go
import "github.com/etkecc/go-kit/template"

out, err := template.Execute("Hi {{.Name}}", map[string]string{"Name": "world"})
// "Hi world"

s := template.May("Hi {{.Name}}", vars)   // returns the raw template if exec fails or comes back empty
s = template.Must("Hi {{.Name}}", vars)   // panics on error; only for compile-time-constant templates
```

## It's `html/template`, and that matters

This wraps `html/template`, **not** `text/template`. So output is HTML-escaped by default, and dropping user-supplied data into a page won't hand you an XSS hole: `<script>` comes out as `&lt;script&gt;` without you lifting a finger.

The flip side, so it doesn't bite you: this escapes for HTML even when you're rendering something that isn't HTML. Build a plaintext email body through here and an `&` in someone's name arrives as `&amp;`. For HTML output that's exactly right; for plaintext, reach for `text/template` directly instead. Right tool, right context.

Three functions, that's the surface. [godoc](https://pkg.go.dev/github.com/etkecc/go-kit/template) has the exact signatures.

## License

GNU LGPL-3.0. See [../LICENSE](../LICENSE).
