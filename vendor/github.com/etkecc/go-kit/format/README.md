# format

[![Go Reference](https://pkg.go.dev/badge/github.com/etkecc/go-kit.svg)](https://pkg.go.dev/github.com/etkecc/go-kit/format)

Markdown to HTML, via [goldmark](https://github.com/yuin/goldmark), tuned for Matrix-adjacent surfaces. One function, `Render`, sane defaults, no ceremony. Inspired by mautrix-go's `format` package.

## It's a separate module (and its Go floor is old on purpose)

`format` has its own `go.mod`, because it depends on goldmark and the root go-kit module refuses to carry dependencies. So you install it on its own line:

```go
go get github.com/etkecc/go-kit/format
```

Here's the deliberate part: its Go floor is **1.22**, lower than the root's 1.26. That's not neglect, it's a lifeline. Some service is stuck on an old Go version and still needs to render Markdown, and pinning the floor low means it can. Don't "helpfully" bump `format/go.mod` up to match the parent without checking who's still down there, or you'll strand exactly the caller this floor exists for.

```go
import "github.com/etkecc/go-kit/format"

html := format.Render("Ship **it** or ~~ship~~ it.")
// "Ship <strong>it</strong> or <del>ship</del> it."
```

## What Render does

- **Single paragraph in, no wrapper out.** A one-paragraph input comes back without the outer `<p>...</p>`, so you can drop it inline. Multi-paragraph input keeps its `<p>` tags. Empty input gives empty output, no stray tags.
- **Every link opens in a new tab.** All `<a>` tags get `target="_blank"` via an AST transformer, which is what you want for user content pasted into a chat.
- **Extensions:** strikethrough and tables.
- **Raw HTML passes through** (goldmark's Unsafe mode) and hard line breaks are honored. That "unsafe" is a real choice: this renders content you already trust or sanitize elsewhere, not arbitrary attacker input straight to a browser. If the source is hostile, sanitize the output before you serve it.

That's the whole package. [godoc](https://pkg.go.dev/github.com/etkecc/go-kit/format) has the exported goldmark handles if you want to build on them.

## License

GNU LGPL-3.0. See [../LICENSE](../LICENSE).
