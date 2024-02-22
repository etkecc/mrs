# [MSC1929](https://github.com/matrix-org/matrix-spec-proposals/blob/main/proposals/1929-admin-contact.md) go client

This library parses MSC1929 support file, and provides a go client to interact with the admin contact API.

Initially it was developed to be used in the [Matrix Rooms Search](https://gitlab.com/etke.cc/mrs/api) project, but it can be used in any other project that needs to interact with the admin contact API.

## Usage

```go
package main

import (
    "fmt"
    "gitlab.com/etke.cc/msc1929"
)

func main() {
    contacts, err := msc1929.Get("matrix.org")
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println(contactts.AdminEmails())
}
```
