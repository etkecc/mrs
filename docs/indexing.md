# How to add your server's public rooms to the index?

How can you add a matrix server to the index on some MRS instance?
Use POST `/discover/{server_name}` endpoint, here is example using the [MatrixRooms.info](https://matrixrooms.info) demo instance and `example.com` homeserver:

```bash
curl -X POST https://apicdn.matrixrooms.info/discover/example.com
```

If your server publishes room directory over federation and has public rooms within the directory,
they will appear in the search index after the next full reindex process (should be run daily).

Please keep in mind that to have a room indexed, you have to:

1. Explicitly mark a room federatable when creating it
2. Explicitly mark a room as public in room settings
3. Explicitly publish a room in the room directory
4. Explicitly publish your room catalog over federation

## Why my server and its public rooms aren't discovered/parsed/included?

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

in case of [etke.cc/ansible](https://github.com/etkecc/ansible) and [matrix-docker-ansible-deploy](https://github.com/spantaleev/matrix-docker-ansible-deploy), add the following to your `vars.yml` configuration file:

```yaml
matrix_synapse_allow_public_rooms_over_federation: true
```
