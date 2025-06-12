# Matrix Rooms Search

A fully-featured, standalone, matrix rooms search service, available both in web (via [HTTP API](./openapi.yml)) and natively in Matrix (via [Matrix Federation API](./docs/integrations.md)).

Dependencies? None.

A public demo instance is available at **[MatrixRooms.info](https://matrixrooms.info)**.

<!-- vim-markdown-toc GitLab -->

* [📌 What MRS Does](#-what-mrs-does)
* [🔍 Why This Exists](#-why-this-exists)
* [🔐 Privacy and Respect](#-privacy-and-respect)
* [❌ Opting Out / Deindexing](#-opting-out-deindexing)
* [How it works?](#how-it-works)
    * [Room configuration](#room-configuration)
    * [Discovery and indexing](#discovery-and-indexing)
    * [How the MSC1929 integration works](#how-the-msc1929-integration-works)
    * [API](#api)
* [Quick Start](#quick-start)
    * [Integrations](#integrations)
* [Support](#support)

<!-- vim-markdown-toc -->

## 📌 What MRS Does

* Queries publicly available Matrix rooms using the documented Matrix federation API:

  ```
  GET /_matrix/federation/v1/publicRooms
  ```
* Displays metadata from this API—such as room name, topic, number of joined users, and aliases.
* Does **not** join rooms, collect messages, user profiles, or private data.
* Does **not** circumvent privacy settings. It only indexes what a homeserver explicitly publishes via federation.

[Protocol documentation](https://spec.matrix.org/latest/server-server-api/#get_matrixfederationv1publicrooms)

## 🔍 Why This Exists

Matrix has no central directory of public rooms. MRS helps solve that by:

* Making public rooms easier to find.
* Improving community visibility across federated homeservers.
* Enabling discovery tools to integrate public room metadata into websites and search platforms.

This can help new users connect with open communities more easily.

## 🔐 Privacy and Respect

MRS is built with privacy and transparency in mind. Here's how it handles data:

| Feature              | Details                                                          |
| -------------------- | ---------------------------------------------------------------- |
| **Only public data** | MRS queries only federation-exposed public room directories.     |
| **No joins**         | MRS does not join rooms or access messages.                      |
| **No profiling**     | MRS does not collect or store user data.                         |
| **Easy opt-out**     | Homeserver admins can remove their content at any time.          |

## ❌ Opting Out / Deindexing

MRS by default respects homeserver admins' decisions to opt out, providing vaiours ways on server and room level.
Please visit [deindexing documentation](./docs/deindexing.md) for more details.

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
