<!--
SPDX-FileCopyrightText: 2023 Nikita Chernyi
SPDX-FileCopyrightText: 2025 Suguru Hirahara

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Bootstrapping

To get started with MRS, you need index some Matrix servers first. As a good starting point, you may use [The-Federation.info](https://the-federation.info) public API to get the first servers.

Follow the example as below to interact with the API:

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
