# Integrations

## [SearXNG](https://docs.searxng.org)

SearXNG is a free internet metasearch engine which aggregates results from more than 70 search services.
Users are neither tracked nor profiled. 
Additionally, SearXNG can be used over Tor for online anonymity.

Just use the [SearXNG docs](https://docs.searxng.org/dev/engines/online/mrs.html).

## [Matrix Federation API](https://spec.matrix.org/latest/)

MRS implements the mandatory subset of Matrix Federation API to provide the Public Rooms Directory over federation,
so you can use it in your matrix client apps directly.

### Element Web/Desktop

1. Click on `Search` (Ctrl+K) in the top-left corner
2. Modal window will be opened, scroll down
3. Click on `Public rooms`
4. Under the search input, click on server selection (`Show: <your server name>`) and click on the `Add new server...`
5. In the opened modal window enter the server name from the config.yml (`matrix.server_name` value).
6. Click on `Add`

## [Synapse](https://matrix-org.github.io/synapse/latest/)

[Synapse module](https://matrix-org.github.io/synapse/latest/modules/writing_a_module.html) not yet implemented
