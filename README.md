# Matrix Rooms Search [![Donate on Liberapay](https://liberapay.com/assets/widgets/donate.svg)](https://liberapay.com/mrs/donate) 

A fully-featured, standalone, matrix rooms search service.


<!-- vim-markdown-toc GitLab -->

* [How it works?](#how-it-works)
    * [Discovery and indexing](#discovery-and-indexing)
        * [How to opt-in?](#how-to-opt-in)
        * [How to opt-out?](#how-to-opt-out)
    * [How the MSC1929 integration works](#how-the-msc1929-integration-works)
        * [How to opt-in?](#how-to-opt-in-1)
        * [How to opt-out?](#how-to-opt-out-1)
    * [API](#api)
* [Quick Start](#quick-start)
    * [Deploy](#deploy)
        * [Manual](#manual)
        * [Ansible](#ansible)
    * [Bootstrapping](#bootstrapping)
    * [Integrations](#integrations)
        * [With SearXNG](#with-searxng)
        * [With Matrix API](#with-matrix-api)
        * [With Synapse](#with-synapse)
* [Troubleshooting](#troubleshooting)
    * [Why my server and its public rooms aren't discovered/parsed/included?](#why-my-server-and-its-public-rooms-arent-discoveredparsedincluded)
    * [Why MRS doesn't contact me when a room from my server is reported.](#why-mrs-doesnt-contact-me-when-a-room-from-my-server-is-reported)
* [Development](#development)
    * [how to add new field](#how-to-add-new-field)

<!-- vim-markdown-toc -->

## How it works?

1. Discover matrix servers (a.k.a find alive and properly configured) from provided config
2. Parse public rooms from the discovered servers
3. Ingest parsed public rooms into search index

Each step can be run separately or all at once using admin API

### Discovery and indexing

#### How to opt-in?

How can you add a matrix server to the index on some MRS instance?
Use POST `/discover/{server_name}` endpoint, here is example using the [MatrixRooms.info](https://matrixrooms.info) demo instance and `example.com` homeserver:

```bash
curl -X POST https://apicdn.matrixrooms.info/discover/example.com
```

If your server publishes room directory over federation and has public rooms within the directory,
they will appear in the search index after the next full reindex process (should be run daily)

#### How to opt-out?

How can you remove your homeserver's rooms from the index?
Just unpublish them or stop publishing room directory over federation.
MRS tries to follow the specification and be polite, so it uses only information that was explicitly published.

### How the [MSC1929](https://github.com/matrix-org/matrix-spec-proposals/pull/1929) integration works

MRS will parse MSC1929 contacts automatically during the discovery phase and store them into db.
When a room is reported using the `/mod/report/{room_id}` endpoint, MRS will check if the room's server
has MSC1929 contacts. If email address(-es) are listed within the contacts, report details will be sent
to the administrators of the Matrix server to which the room belongs.

#### How to opt-in?

Add `/.well-known/matrix/support` file with the following structure:

```json
{
  "contacts": [
    {
      "email_address": "your@email.here",
      "matrix_id": "@your:mxid.here"
    }
  ]
}
```
File must be served on the homeserver name domain (`@you:example.com` -> `https://example.com/.well-known/matrix/support`)

At this moment, MRS works with emails only

#### How to opt-out?

Just unlist your email from the MSC1929 file

### API

Check [openapi.yml](./openapi.yml)

## Quick Start

### Deploy

#### Manual

1. Build mrs
2. Copy `config.yml.sample` into `config.yml` and adjust it
3. Run mrs with `-c config.yml`
4. You probably want to call `/-/full` admin API endpoint at start

#### Ansible

MRS is fully integrated into the [MASH Playbook](https://github.com/mother-of-all-self-hosting/mash-playbook/),
just use the [playbook docs](https://github.com/mother-of-all-self-hosting/mash-playbook/blob/main/docs/services/mrs.md).

### Bootstrapping

To get started with MRS, you need index some matrix servers first.
As a good starting point, you may use [The-Federation.info](https://the-federation.info) public API to get the first servers.

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

### Integrations

#### With [SearXNG](https://docs.searxng.org)

SearXNG is a free internet metasearch engine which aggregates results from more than 70 search services.
Users are neither tracked nor profiled. 
Additionally, SearXNG can be used over Tor for online anonymity.

Just use the [SearXNG docs](https://docs.searxng.org/dev/engines/online/mrs.html).

#### With [Matrix API](https://spec.matrix.org/latest/)

[Matrix Server-Server API endpoint](https://spec.matrix.org/v1.8/server-server-api/#public-room-directory) not yet implemented

#### With [Synapse](https://matrix-org.github.io/synapse/latest/)

[Synapse module](https://matrix-org.github.io/synapse/latest/modules/writing_a_module.html) not yet implemented

## Troubleshooting

### Why my server and its public rooms aren't discovered/parsed/included?

Your server must publish public rooms over federation (`/_matrix/federation/v1/publicRooms` endpoint), eg: `https://matrix.etke.cc:8448/_matrix/federation/v1/publicRooms`

**I get error on public rooms endpoint**, something like:

```json
{"errcode":"M_FORBIDDEN","error":"You are not allowed to view the public rooms list of example.com"}
```

In that case you should adjust your server's configuration.
For synapse, you need to add the following config options in the `homeserver.yaml`:

```yaml
allow_public_rooms_over_federation: true
```

in case of [etke.cc/ansible](https://gitlab.com/etke.cc/ansible) and [mdad](https://github.com/spantaleev/matrix-docker-ansible-deploy), add the following to your vars.yml:

```yaml
matrix_synapse_allow_public_rooms_over_federation: true
```

### Why MRS doesn't contact me when a room from my server is reported.

You have to serve the `/.well-known/matrix/support` file with at least 1 email in it.

in case of [etke.cc/ansible](https://gitlab.com/etke.cc/ansible) and [mdad](https://github.com/spantaleev/matrix-docker-ansible-deploy), add the following to your vars.yml:

```yaml
matrix_well_known_matrix_support_enabled: true
matrix_homeserver_admin_contacts:
  - matrix_id: "@you:example.com" # optional, remove if not needed
    email_address: "you@example.com" # required for MRS MSC1929 integration
matrix_homeserver_support_url: "https://example.com/help" # optional, remove if not needed
```

## Development

### how to add new field

1. adjust `model/matrix.go` and `model/search.go` if needed
2. adjust `repository/search/bleve.go` `getIndexMapping()`
3. adjust `repository/search/search.go` `parseSearchResults()`
4. adjust `services/search.go` `getSearchQuery()`
