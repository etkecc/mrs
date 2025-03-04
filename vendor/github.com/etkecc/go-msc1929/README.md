# Matrix Server Contacts go client

This small library provides a go client to interact with [`/.well-known/matrix/support`](https://spec.matrix.org/latest/client-server-api/#getwell-knownmatrixsupport) endpoint.

The following MSCs are supported:

* [MSC1929](https://github.com/matrix-org/matrix-spec-proposals/pull/1929) - the `/.well-known/matrix/support` endpoint
* [MSC4121](https://github.com/matrix-org/matrix-spec-proposals/pull/4121) - the moderator role
* [MSC4265](https://github.com/matrix-org/matrix-spec-proposals/pull/4265) - the DPO role

Initially it was developed to be used in the [Matrix Rooms Search](https://github.com/etkecc/mrs) project, but it can be used in any other project that needs to interact with the admin contact API.

## Usage

```go
package main

import (
    "fmt"
    "github.com/etkecc/go-msc1929"
)

func main() {
    contacts, err := msc1929.Get("matrix.org")
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println(contacts.AdminEmails())
}
```
