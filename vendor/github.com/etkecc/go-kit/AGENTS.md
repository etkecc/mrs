# go-kit

Go 1.26 utility library used by etkecc backends; three Go modules: dependency-free root, `format` (Go 1.22, goldmark), `crypter/yaml` (Go 1.25, yaml.v3).

## Commands (run from the repo root)

- Test: `just test` # root module with coverage
- Nested modules: `go -C format test ./...` and `go -C crypter/yaml test ./...`
- Single test: `go test . -run '^TestUniq$'`
- Lint: `just lint` # root module; comment guard covers tracked files
- Build: `go build ./...` # root module
- Full check: `just lint && just test && go test -race ./... && go -C format test ./... && go -C crypter/yaml test ./...`
- Prerequisites: Go 1.26 or newer; just; golangci-lint built with a Go release compatible with the toolchain, or typecheck fails with export data version errors.

## Conventions

- Root module is stdlib-only; a helper that needs a dependency becomes its own nested module.
- The three modules install separately; keep their `go.mod` files independent.
- Stateful types are constructor-built (`New*`); a zero value is unusable unless its doc says otherwise.
- Library code returns errors; panic only in `Must*` helpers and hardcoded-constant parsing.

## Landmines

- Root `go test ./...`, `go build ./...`, and `just test` only cover the root module; `format/` and `crypter/yaml/` are separate modules.

## Boundaries

- Always: `just lint && just test && go test -race ./... && go -C format test ./... && go -C crypter/yaml test ./...` before handoff.
- Ask first: `just update` (root module only); adding a dependency.
- Never: add a dependency to root `go.mod`; merge `format` or `crypter/yaml` into the root module.

## Pointers

- `README.md` - module split and package behavior. Read before changing exported API or module boundaries.
- `crontab/AGENTS.md` - cron schedule and DST rules. Read before editing `crontab/`.
- `crypter/AGENTS.md` - AES-GCM tag and key contract. Read before editing `crypter/`.
- `crypter/yaml/AGENTS.md` - yaml module contract and walk rules. Read before editing `crypter/yaml/`.
- `format/AGENTS.md` - format module contract and Go floor. Read before editing `format/`.
- `httpclient/AGENTS.md` - constructor presets, retries, and SSRF guard. Read before editing `httpclient/`.
- `migrater/AGENTS.md` - migration ordering and drift rules. Read before editing `migrater/`.
- `retry/AGENTS.md` - backoff and `DelayHinter` contract. Read before editing `retry/`.
- `template/AGENTS.md` - HTML escaping and fallback behavior. Read before editing `template/`.
- `workpool/AGENTS.md` - single-use pool and queue rules. Read before editing `workpool/`.
