# Bootstrapping

To get started with MRS, you need index some matrix servers first.
As a good starting point, you may use [The-Federation.info](https://the-federation.info) public API to get the first servers.

```bash
curl 'https://the-federation.info/v1/graphql' \
    -X POST \
    -H 'content-type: application/json' \
    --data '{
        "query":"query MatrixServers { thefederation_node( where: {blocked: {_eq: false}, thefederation_platform: {id: {_eq: 41}}} order_by: {last_success: desc} ) { host }}",
        "variables":null,
        "operationName":"MatrixServers"
    }' | jq -r '.data.thefederation_node[] | "- " + .host'
```
