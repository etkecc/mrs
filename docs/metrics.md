# Stats and metrics

MRS collects stats and metrics and exposes them on the following endpoints:

* `/stats` - publicly available stats, contains only basic stats
* `/-/status` - private admin endpoint, contains full stats about discovered, parsed and indexed servers and rooms
* `/metrics` - private metrics endpoint, using the Prometheus metrics format

## Servers

Servers-related stats

### Online

* `servers` and `details.servers.online` on `/stats`
* `servers.online` on `/-/status`
* `mrs_servers_online` on `/metrics`

The total amount of online and federatable matrix servers discovered. Updated during the discovering phase

### Indexable

* `details.servers.indexable` on `/stats`
* `servers.indexable` on `/-/status`
* `mrs_servers_indexable` on `/metrics`

The total amount of online federatable matrix servers which publish rooms directory over federation. Updated during the discovery phase

### Blocked

* not presented on `/stats`
* `servers.blocked` on `/-/status`
* not presented on `/metrics`

The amount of servers in the config (config.yml `blocklist.servers`)

## Rooms

### Indexed

* `rooms` and `details.rooms.indexed` on `/stats`
* `rooms.indexed` on `/-/status`
* `mrs_rooms_indexed` on `/metrics`

The total amount of indexed (searchable) rooms

### Parsed

* `details.rooms.parsed` on `/stats`
* `rooms.parsed` on `/-/status`
* `mrs_rooms_indexed` on `/metrics`

The total amount of rooms parsed from public rooms directories

### Blocked

* not presented on `/stats`
* `rooms.blocked` on `/-/status`
* not presented on `/metrics`

The total amount of banned rooms

### Reported

* not presented on `/stats`
* `rooms.reported` on `/-/status`
* not presented on `/metrics`

The total amount of reported rooms

## Search

### Queries

* not presented on `/stats`
* not presented on `/-/status`
* `mrs_search_queries` on `/metrics`

The total amount of search requests done
