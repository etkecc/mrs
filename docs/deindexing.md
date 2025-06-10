# How to deindex/unlist/unpublish/block/remove your server's rooms from index

If your server publishes room directory over federation and has public rooms within the directory,
they will appear in the search index after the next full reindex process (should be run daily).

Please keep in mind that to have a room indexed, you have to:

1. Explicitly mark a room federatable when creating it
2. Explicitly mark a room as public in room settings
3. Explicitly publish a room in the room directory
4. Explicitly publish your room catalog over federation

When you do all of those steps, you clearly understand the consequences of your decisions,
i.e. a particular room that was made public, federatable,
published in room catalog and then shared room catalog over federation will be accessed over federation.

MRS can't index a room if you didn't explicitly allow a room to be visible over federation.
You either publish something over federation, or not.
MRS is not a special thing, it uses the same API and the same set of rules as any other matrix server does.

## unpublish your room directory from the federation

If your server is indexed, that means you explicitly published your public rooms directory over federation.
If you don't like that the information you explicitly published over federation is accessed over federation,
you should consider unpublishing it.

For synapse, you need to add the following config options in the `homeserver.yaml`:

```yaml
allow_public_rooms_over_federation: false
```

in case of [etke.cc/ansible](https://github.com/etkecc/ansible) and [matrix-docker-ansible-deploy](https://github.com/spantaleev/matrix-docker-ansible-deploy), add the following to your `vars.yml` configuration file:

```yaml
matrix_synapse_allow_public_rooms_over_federation: false
```

However, if you think that "publishing over federation, but not for that particular member of the federation" segregation is a good thing
for Matrix protocol, MRS has several options to unlist/unpublish/block/remove your server and its rooms from indexing.

## (specific room) using room topic

As a room administrator/moderator, you could add special string to the room topic to prevent indexing:

```
(MRS-noindex:true-MRS)
```

[More details about room configuration](./room-configuration.md)

## contact instance maintainers

MRS is open source project (code), it doesn't parse/index/process any data by itself, so you have to contact specific instance's maintainers.
Each instance should have a page with details and contacts.

That method is discouraged, because it requires manual intervention by instance's maintainers, and thus each request processing will take way more time than using any of the methods described above.
