# Matrix Rooms Search [![Donate on Liberapay](https://liberapay.com/assets/widgets/donate.svg)](https://liberapay.com/mrs/donate) 

A fully-featured, standalone, matrix rooms search service.

<!-- vim-markdown-toc GitLab -->

* [How it works?](#how-it-works)
    * [Discovery and indexing](#discovery-and-indexing)
    * [How the MSC1929 integration works](#how-the-msc1929-integration-works)
    * [API](#api)
* [Quick Start](#quick-start)
    * [Integrations](#integrations)

<!-- vim-markdown-toc -->

## How it works?

1. Discover matrix servers (a.k.a find alive and properly configured) from provided config
2. Parse public rooms from the discovered servers
3. Ingest parsed public rooms into search index

Each step can be run separately or all at once using admin API

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
