# Go Bindle Client

[![Go Reference](https://pkg.go.dev/badge/github.com/deislabs/go-bindle.svg)](https://pkg.go.dev/github.com/deislabs/go-bindle)

A [Bindle](https://github.com/deislabs/bindle) client for Go.

## Using the Client

Below is a simple example using the client to get an invoice

```go
package main

import (
    "fmt"

    "github.com/deislabs/go-bindle/client"
)

func main() {
    // The second parameter takes an optional tls.Config if you have any special TLS configuration 
    // needs. A nil config will just use the default
    bindleClient, err := client.New("https://my.bindle.server.com/v1", nil)
    if err != nil {
        panic(err)
    }

    invoice, err := bindleClient.GetInvoice("example.com/foo/1.0.0")
    if err != nil {
        panic(err)
    }

    fmt.Println(invoice)
}
```

Please visit the [documentation](https://pkg.go.dev/github.com/deislabs/go-bindle) for more
information on each of the functions

## Contributing

We welcome any contributions or feedback! If you'd like to contribute code, please open a [Pull
Request](https://github.com/deislabs/go-bindle/pulls).

This project has adopted the [Microsoft Open Source Code of
Conduct](https://opensource.microsoft.com/codeofconduct/).

For more information see the [Code of Conduct
FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact
[opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

### Tests

Testing for the client is mostly done with integration tests. In order to run the integration tests,
you'll need a [`bindle-server`
binary](https://github.com/deislabs/bindle/tree/master/docs#from-canary-builds) available for the
tests to use. By default, the tests will search in your `$PATH` for an executable called
`bindle-server`. You can also configure it to use a specific binary by setting the
`BINDLE_SERVER_PATH` environment variable. Currently the testing pipeline uses the canary
`bindle-server` for testing and will continue to do so until the spec and project have stabilized
