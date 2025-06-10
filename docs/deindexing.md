<!--
SPDX-FileCopyrightText: 2023 - 2024 Nikita Chernyi
SPDX-FileCopyrightText: 2025 Suguru Hirahara

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Deindexing: How to block your server's rooms from being indexed

If your server publishes room directory over federation and its public rooms are listed on the directory, they will be included in the search index with daily full reindexing process. See [this page](indexing.md) for details about indexing.

There are several ways to prevent rooms from being indexed. You can choose any of those methods per your needs and preference.

**Note**: if a room is public, federatable, and published on the room catalog which is shared over the federation, the room can be freely discovered and accessed over federation, whether by a MRS instance or not. **If the room is not accessible over federation, MRS does not add the room to the index**, strictly following and respecting the Matrix protocol.

## Methods to prevent rooms from being indexed

### Unpublish your room directory from the federation

For indexing rooms, MRS only makes use of information on your public room directory which has been explicitly published on your homeserver. Note that the information is publicly available and basically anyone can retrieve that information.

Therefore, it is expected protocol-wise to unpublish the room directory in order to prevent such information from being accessed altogether.

For Synapse homeserver, add the following config options in the `homeserver.yaml`:

```yaml
allow_public_rooms_over_federation: false
```

If you use [etke.cc/ansible](https://github.com/etkecc/ansible) and [matrix-docker-ansible-deploy](https://github.com/spantaleev/matrix-docker-ansible-deploy) to manage your Matrix homeserver, add the following to your `vars.yml` configuration file:

```yaml
matrix_synapse_allow_public_rooms_over_federation: false
```

As enabling the option unfederates your homeserver, *nobody* outside of the homeserver, including a MRS instance, can access to it over federation, making it technically impossible to index the rooms as a matter of course.

### Add special strings to the room's topic

If you do want to make the server itself federated and the room directory to be published over the federation, only preventing rooms from being indexed, you can add special strings which instruct MRS instances not to index them.

As a room administrator/moderator, you can add these strings to the room topic to prevent indexing:

```
(MRS-noindex:true-MRS)
```

See [this page](./room-configuration.md) for details about room configuration.

### Contact MRS instance maintainers

If none of the methods described above is available (such as due to lack of permissions to edit the room's topic), you might try contacting to MRS instance's maintainers. On each instance should there be a page with contact details.

We do not recommend this method, as it requires manual intervention by instance's maintainers, and thus it will take much more time for each request to be handled than those methods.

⚠️ **Note**: MRS is open source project (code) which enables to set up MRS instances. *It is these instances which index rooms, not the project*, therefore it is not able for the team behind the project to deindex rooms from MRS instances.
