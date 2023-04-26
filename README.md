# Matrix Rooms Search

A fully-featured, standalone, matrix rooms search service.

## How it works?

1. Discover matrix servers (a.k.a find alive and properly configured) from provided config
2. Parse public rooms from the discovered servers
3. Ingest parsed public rooms into search index

Each step can be run separately or all at once using admin API

### API

Check [openapi.yml](./openapi.yml)

## Quick Start

1. Build mrs
2. Copy `config.yml.sample` into `config.yml` and adjust it
3. Run mrs with `-c config.yml`
4. You probably want to call `/-/full` admin API endpoint at start

## Development

### how to add new field

1. adjust `model/matrix.go` and `model/search.go` if needed
2. adjust `repository/search/bleve.go` `getIndexMapping()`
3. adjust `repository/search/search.go` `parseSearchResults()`
4. adjust `services/search.go` `getSearchQuery()`

## Troubleshooting

### Why my server and its public rooms aren't discovered/parsed/included?

1. Your server must have valid `/.well-known/matrix/client`, e.g. `https://etke.cc/.well-known/matrix/client`
2. Your server must publish public rooms over federation without auth (`/_matrix/client/v3/publicRooms` endpoint), eg: `https://matrix.etke.cc/_matrix/client/v3/publicRooms`

**I get error on public rooms endpoint**, something like:

```json
{"errcode":"M_MISSING_TOKEN","error":"Missing access token"}
```

In that case you should adjust your server's configuration.
For synapse, you need to add the following config options in the `homeserver.yaml`:

```yaml
allow_public_rooms_over_federation: true
allow_public_rooms_without_auth: true
```

in case of [etke.cc/ansible](https://gitlab.com/etke.cc/ansible) and [mdad](https://github.com/spantaleev/matrix-docker-ansible-deploy), add the following to your vars.yml:

```yaml
matrix_synapse_allow_public_rooms_over_federation: true
matrix_synapse_allow_public_rooms_without_auth: true
```

### Where can I get list of servers?

**get list of known servers from own matrix server db**

```bash
psql -d synapse -c "select '- '||destination from destinations;" > destinations.txt
```

**get list of known servers from [the-federation.info](https://the-federation.info)**

```bash
curl 'https://the-federation.info/v1/graphql' \
    -X POST \
    -H 'content-type: application/json' \
    --data '{
        "query":"query MatrixServers { thefederation_node( where: {blocked: {_eq: false}, thefederation_platform: {id: {_eq: 41}}} order_by: {last_success: desc} ) { host }}",
        "variables":null,
        "operationName":"MatrixServers"
    }' | jq -r '.data.thefederation_node[] | "- " + .host'
```
