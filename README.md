<!--
SPDX-FileCopyrightText: 2023 - 2025 Nikita Chernyi
SPDX-FileCopyrightText: 2025 Suguru Hirahara

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Matrix Rooms Search

**Matrix Rooms Search** (short: MRS) is a fully-featured, standalone, Matrix rooms search service, available both in web (via [HTTP API](./openapi.yml)) and natively in Matrix (via [Matrix Federation API](./docs/integrations.md)).

No dependencies are required to run a Matrix Rooms Search instance.

At **[MatrixRooms.info](https://matrixrooms.info)** is a public demo instance available.

<!-- vim-markdown-toc GFM -->

* [📌 What MRS Does](#-what-mrs-does)
* [🔍 Why This Exists](#-why-this-exists)
* [🔐 Privacy and Respect](#-privacy-and-respect)
* [❌ Opting Out / Deindexing](#-opting-out--deindexing)
* [Mechanism](#mechanism)
    * [Room configuration](#room-configuration)
    * [Discovery and indexing](#discovery-and-indexing)
    * [MSC1929 integration](#msc1929-integration)
    * [API](#api)
* [Quick Start](#quick-start)
    * [Integrations](#integrations)
* [Support](#support)

<!-- vim-markdown-toc -->

## 📌 What MRS Does

* Queries only federation-exposed public room directories with the Matrix federation API:

  ```
  GET /_matrix/federation/v1/publicRooms
  ```
* Displays metadata retrieved via the API — such as room name, topic, number of joined users, and aliases.

See [the protocol documentation](https://spec.matrix.org/latest/server-server-api/#get_matrixfederationv1publicrooms) for technical details.

### What MRS does *not* do

* MRS does **not** join rooms, collect messages, user profiles, or private data.
* MRS does **not** circumvent privacy settings. It only indexes what a homeserver explicitly publishes via federation.

## 🔍 Why This Exists

Matrix has inherently no central directory of public rooms, which makes it difficult for visitors to find ones they might be interested in and for room's administrators to advertise their communities to the rest of the Matrix ecosystem.

MRS helps to solve the difficulty by:

* Making *public rooms* easier to find.
* Improving *community visibility* across federated homeservers.
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

MRS only makes use of publicly-available information. No privacy-invasive method is deployed.

## ❌ Opting Out / Deindexing

Following the Matrix protocol, MRS fully respects homeserver admins' choice to prohibit MRS instances from indexing their public rooms. Please visit [deindexing documentation](./docs/deindexing.md) for more details.

## Mechanism

MRS essentially works as below:

1. Discover matrix servers (a.k.a find alive and properly configured) from provided config
2. Parse public rooms from the discovered servers
3. Ingest parsed public rooms into search index

Each step can be run separately or all at once using admin API.

### Room configuration

MRS allows you to configure different room parameters by adding special configuration strings to the room topic/description. Check [room-configuration.md](./docs/room-configuration.md) for details.

### Discovery and indexing

**Opt-in**: check the [docs/indexing.md](./docs/indexing.md)

**Opt-out**: check the [docs/deindexing.md](./docs/deindexing.md)

### [MSC1929](https://github.com/matrix-org/matrix-spec-proposals/pull/1929) integration

MRS integrates [MSC1929](https://github.com/matrix-org/matrix-spec-proposals/pull/1929) natively, in order to help homeserver administrators to combat unlawful actions. See details on [this page](./msc1929.md).

### API

See [openapi.yml](./openapi.yml) for details about the API.

## Quick Start

See [docs/deploy.md](./docs/deploy.md) and [docs/bootstrapping.md](./docs/bootstrapping.md) for details.

MRS is integrated with the [MASH playbook](https://github.com/mother-of-all-self-hosting/mash-playbook/), which helps you to run various web services with Ansible. Check [its documentation](https://github.com/mother-of-all-self-hosting/mash-playbook/blob/main/docs/services/mrs.md) for details about running a MRS instance on your server.

### Integrations

MRS provides integrations for Matrix clients and other applications like [SearXNG](https://docs.searxng.org). See [docs/integrations.md](./docs/integrations.md) for details.

## Support

If you have questions, please come to our support room at [#mrs:etke.cc](https://matrixrooms.info/room/mrs:etke.cc) and do not hesitate to ask them!
