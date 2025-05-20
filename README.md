# Matrix Rooms Search

A fully-featured, standalone, matrix rooms search service, available both in web (via [HTTP API](./openapi.yml)) and natively in Matrix (via [Matrix Federation API](./docs/integrations.md)).

Dependencies? None.

<!-- vim-markdown-toc GitLab -->

* [How it works?](#how-it-works)
    * [Room configuration](#room-configuration)
    * [Discovery and indexing](#discovery-and-indexing)
    * [How the MSC1929 integration works](#how-the-msc1929-integration-works)
    * [API](#api)
* [Quick Start](#quick-start)
    * [Integrations](#integrations)
* [Support](#support)

<!-- vim-markdown-toc -->

## How it works?

1. Discover matrix servers (a.k.a find alive and properly configured) from provided config
2. Parse public rooms from the discovered servers
3. Ingest parsed public rooms into search index

Each step can be run separately or all at once using admin API

### Room configuration

MRS allows you to configure different room parameters by adding special configuration strings to the room topic/description.
Check [room-configuration.md](./docs/room-configuration.md).

### Discovery and indexing

**Opt-in**: check the [docs/indexing.md](./docs/indexing.md)

**Opt-out**: check the [docs/deindexing.md](./docs/deindexing.md)

### How the [MSC1929](https://github.com/matrix-org/matrix-spec-proposals/pull/1929) integration works

Check the [docs/msc1929.md](./docs/msc1929.md)

### API

Check [openapi.yml](./openapi.yml)

## Quick Start

Check [docs/deploy.md](./docs/deploy.md) and [docs/bootstrapping.md](./docs/bootstrapping.md)

### Integrations

Check [docs/integrations.md](./docs/integrations.md)

## Support

[#mrs:etke.cc](https://matrix.to/#/#mrs:etke.cc) matrix room
