# template

Thin html/template wrapper in the root module; `Execute` returns errors, `May` falls back, `Must` panics.

## Contract

- `Execute` parses and runs `tplString` with `vars` as dot; a parse or execute failure returns `""` and the error.
- `May` returns the raw template string when execution fails or renders empty; errors are swallowed.
- `Must` panics on any error; use it only for compile-time-constant templates.
- Output is HTML-escaped because this wraps html/template.

## Landmines

- Rendering plaintext through this package HTML-escapes it (`&` becomes `&amp;`); use text/template for non-HTML output.
- `May` cannot distinguish an intentional empty render from a failure; it returns the raw template in both cases.

## Verify

`go test ./template`

## Pointers

- `README.md` - three functions and the HTML escaping rationale.
- Root rules: `../AGENTS.md`.
