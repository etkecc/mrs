<!--
SPDX-FileCopyrightText: 2023 - 2024 Nikita Chernyi
SPDX-FileCopyrightText: 2025 Suguru Hirahara

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Deindexing: How to block your server's rooms from being indexed

If your server publishes room directory over federation and its public rooms are listed on the directory, they will be included in the search index by Matrix Rooms Search (short: MRS) instances with daily full reindexing process.

## (How indexing occurs)

The rooms will be included in the search index, if these conditions are met:

- The room was configured as federatable when you created it
- The room is set to "public" in room settings
- The room is published on your room directory
- The room catalog is published over federation

⚠️ **Note**: If those conditions are met, any rooms can be discovered and accessed over federation, *whether by a MRS instance or not*. Please note that MRS instances will index rooms purely by following and respecting the Matrix protocol. **They will never index rooms otherwise.** See [this page](indexing.md) for relevant similar information.

## How to prevent rooms from being indexed

There are several ways to prevent rooms from being indexed. You can choose any of those methods per your needs and preference.

### Unpublish your room directory from the federation

For indexing rooms, MRS only makes use of information on your public room directory which has been explicitly published by your homeserver. Note that the information is publicly available and basically any other Matrix server can retrieve that information.

Therefore, it is expected protocol-wise to unpublish the room directory in order to prevent such information from being accessed altogether.

For Synapse homeserver, add the following config options in the `homeserver.yaml`:

```yaml
allow_public_rooms_over_federation: false
```

If you use [etke.cc/ansible](https://github.com/etkecc/ansible) and [matrix-docker-ansible-deploy](https://github.com/spantaleev/matrix-docker-ansible-deploy) to manage your Matrix homeserver, add the following to your `vars.yml` configuration file:

```yaml
matrix_synapse_allow_public_rooms_over_federation: false
```

As enabling the option prevents your homeserver from publishing its public rooms directory over federation, *no other Matrix server*, including MRS instances, can access the directory, making it technically impossible to view/parse/index the public rooms as a matter of course.

### Add special strings to the room's topic

If you do want to make your server publish the public rooms directory over the federation, only preventing rooms from being indexed by MRS instances specifically, you can add special string which instruct any MRS instance not to index them.

As a room administrator/moderator, you can add that string to the room topic to prevent indexing:

```
(MRS-noindex:true-MRS)
```

See [this page](./room-configuration.md) for details about room configuration.

### Contact MRS instance maintainers

If none of the methods described above is available (such as due to lack of permissions to edit the room's topic), you might try contacting to MRS instance's maintainers. On each instance should there be a page with contact details.

We do not recommend this method, as it requires manual intervention by instance's maintainers, and thus it will take much more time for each request to be handled than those methods.

⚠️ **Note**: MRS is open source project (code) which enables to set up MRS instances. *It is these instances which index rooms, not the project*, therefore it is not able for the team behind the project to deindex rooms from MRS instances.
