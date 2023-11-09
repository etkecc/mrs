# Integrations

## Matrix Federation API

MRS implements the mandatory subset of Matrix Federation API to provide the Public Rooms Directory over federation,
so you can use it in your matrix client apps directly.

### Element Web/Desktop

**Temporary** (just for your current session)

1. Click on `Search` (Ctrl+K) in the top-left corner
2. Modal window will be opened, scroll down
3. Click on `Public rooms`
4. Under the search input, click on server selection (`Show: <your server name>`) and click on the `Add new server...`
5. In the opened modal window enter the server name from the config.yml (`matrix.server_name` value).
6. Click on `Add`

**Persistent** (for users of the Element Web/Desktop app)

Add the following to the Element's `config.json`:

```json
"room_directory": {
    "servers": ["matrixrooms.info"]
}
```

If you use [etke.cc/ansible](https://gitlab.com/etke.cc/ansible) or [mdad](https://github.com/spantaleev/matrix-docker-ansible-deploy), add the following to your vars.yml:

```yaml
matrix_client_element_room_directory_servers: ['matrixrooms.info']
```

## [SearXNG](https://docs.searxng.org)

SearXNG is a free internet metasearch engine which aggregates results from more than 70 search services.
Users are neither tracked nor profiled. 
Additionally, SearXNG can be used over Tor for online anonymity.

Just use the [SearXNG docs](https://docs.searxng.org/dev/engines/online/mrs.html).

## MSCs

MSC stands for Matrix Spec Change - a proposed changes to the matrix protocol, but not yet included within it.

### MSC1929

Details: [docs/msc1929.md](./msc1929.md)

### MSC3266

Room preview API, available on `GET /_matrix/client/unstable/im.nheko.summary/summary/{room_id_or_alias}` endpoint, more details in the API spec file
