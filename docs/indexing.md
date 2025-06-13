<!--
SPDX-FileCopyrightText: 2023 Nikita Chernyi
SPDX-FileCopyrightText: 2025 Suguru Hirahara

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Indexing: How to let your server's public rooms added to indexes

To let your Matrix server's public rooms indexed on Matrix Rooms Search instances, you can use the POST `/discover/{server_name}` endpoint following the example below.

```bash
curl -X POST https://api.matrixrooms.info/discover/example.com
```

In the example, the [MatrixRooms.info](https://matrixrooms.info) demo Matrix Rooms Search instance and `example.com` homeserver are specified. Please change them as needed.

## How indexing occurs

If your server publishes room directory over federation and its public rooms are listed on the directory, they will be included in the search index by MRS instances with daily full reindexing process.

The rooms will be included in the search index, if these conditions are met:

- The room was configured as federatable when you created it
- The room is set to "public" in room settings
- The room is published on your server's public rooms directory
- Your server published the public rooms directory over federation

## FAQ

### Why my server and its public rooms are not discovered or included in the indexes?

It is because not all of the conditions described above are met.

In addition to editing the room's configuration, please also make sure that your server publishes the public rooms directory over federation.

For Synapse homeserver, you need to add the following config options in the `homeserver.yaml`:

```yaml
allow_public_rooms_over_federation: true
```

If you use [etke.cc/ansible](https://github.com/etkecc/ansible) and [matrix-docker-ansible-deploy](https://github.com/spantaleev/matrix-docker-ansible-deploy) to manage your Matrix homeserver, add the following to your `vars.yml` configuration file:

```yaml
matrix_synapse_allow_public_rooms_over_federation: true
```

### How can I block my server's rooms from being indexed?

You can check [this page](deindexing.md) for details.
