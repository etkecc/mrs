# format

Separate module github.com/etkecc/go-kit/format: Markdown to HTML via goldmark; the root module stays dependency-free.

## Contract

- Go floor 1.22 on purpose; services on old Go depend on it. Ask before bumping.
- goldmark is the only dependency; keep this module separate from root.
- `Render` drops the `<p>` wrapper for single-paragraph input, opens every link in a new tab, and passes raw HTML through.
- Public surface is `Render` plus the exported goldmark handles.

## Landmines

- Raw HTML passthrough turns hostile Markdown into hostile HTML; sanitize before serving untrusted input.
- The 1.22 floor and goldmark version are compatibility promises; ask before bumping either, or old-Go consumers are stranded.

## Verify

From the repo root: `go -C format test ./...`

## Pointers

- `README.md` - render behavior and the go-floor rationale.
- Root rules: `../AGENTS.md`.
