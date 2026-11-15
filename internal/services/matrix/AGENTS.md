# internal/services/matrix

Matrix federation and client-server surface for indexed rooms; controllers, the crawler, and moderation call it.

## Contract

- When the SRV target differs from the delegated host, the URL, Host, and SNI keep the delegated name; only the TCP dial is pinned to the resolved IP.
- `QueryServerName` returns `("", nil)` on a negative-cache hit; callers treat an empty name as a failed lookup.
- `lookupKeys` rejects a key response unless the server named in the response signs it; auth and notary callers reject a response whose name differs from the request.

## Landmines

- `ValidateAuth` skips signature validation entirely when `matrix.server_name` is `localhost`; deployed configs need their real federation name.
- `ValidateAuth` drains `r.Body`; a handler that reads the body afterwards gets an empty stream. Buffer and restore it first, as `internal/controllers/matrix_s2s.go` does.
- A user-supplied `via` must go through `resolveCSURL`; `QueryCSURL` caches its input, so using it there lets an attacker poison `curlsCache`.
- `GetClientMediaThumbnail` is marked deprecated but `internal/controllers/avatar.go` calls it first; remove both together.
- `dcrURL` discovery is fire-and-forget with `context.WithoutCancel`, bounded by `workers.discovery`; it outlives the request, so request-scoped values are unsafe there.
- `QueryServersKeys` detaches the context once before the workpool; reassigning it inside a worker races under `-race`.

## Verify

`just test ./internal/services/matrix`

## Pointers

- Root rules: `../../../AGENTS.md`.
