# The `/stats` endpoint

When MRS instance discovers and indexes Matrix rooms,
it collects various technical details about the Matrix servers that publish these rooms over federation.

While that information is primarily used for Matrix protocol communication purposes,
it is also may be interesting to see some aggregated statistics about the Matrix Federation observed by this MRS instance.

<!-- vim-markdown-toc GFM -->

* [WARNING](#warning)
* [The endpoint](#the-endpoint)
    * [Servers](#servers)
    * [Rooms](#rooms)
    * [Timeline](#timeline)

<!-- vim-markdown-toc -->

## WARNING

Please, read this carefully:

1. MRS project itself does **NOT** collect, aggregate or publish any information or statistics.
   Each MRS instance is independent and self-hosted by its owner.
2. Stats collected by **a specific instance** do **NOT** represent the whole Matrix Federation.
   They represent only the subset of Matrix servers that are visible by a **specific** MRS instance,
   which may be a small fraction of the whole Matrix Federation.
3. MRS instances do **NOT** share any information or statistics with each other or with any third parties.
   Each MRS instance exposes its stats only on its own endpoints, and not beyond that.
4. The stats exposed by MRS instances **may be incomplete or inaccurate**,
   due to the nature of Matrix Federation and the limitations of MRS itself.
5. Neither MRS project itself nor any of its instances can be considered as an authoritative source of information about the Matrix Federation.
6. More details are listed in the project's readme.

Please, keep these points in mind when interpreting the stats exposed by MRS instances.

## The endpoint

MRS exposes the collected stats on the public `/stats` endpoint.
Here is the list of available stats:

### Servers

* **Online servers** - the total amount of online and federatable Matrix servers discovered.
  Updated during the discovering phase.
    * `/stats`: `servers` and `details.servers.online`
* **Indexable servers** - the total amount of online federatable Matrix servers which publish rooms directory over federation.
  Updated during the discovery phase.
    * `/stats`: `details.servers.indexable`
* **Servers software** - the map of server software (e.g., `synapse`) to the amount of **online** servers running it,
  included only software with at least 1% of the total amount of **online** servers, other software is grouped under `other`.
    * `/stats`: `details.servers.software`

### Rooms

* **Indexed rooms** - the total amount of indexed (searchable) rooms.
  Updated during the indexing phase.
    * `/stats`: `rooms` and `details.rooms.indexed`
* **Parsed rooms** - the total amount of rooms discovered during the parsing phase (not necessarily indexed).
  This number is always greater than or equal to the number of indexed rooms.
    * `/stats`: `details.rooms.parsed`

### Timeline

Apart from the current stats (`details` top-level field), MRS also exposes historical timeline of the same stats,
using the `timeline` top-level field.
The timeline contains the stats snapshot for each full reindexing cycle, but with the following limitations:

* Current month: daily snapshots
* Previous months of the current year: weekly snapshots
* Previous years: monthly snapshots

Each snapshot contains the same stats as the current stats, but representing the state at the time of the snapshot.
