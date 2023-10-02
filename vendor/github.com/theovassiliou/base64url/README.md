# base64url

base64url, implemented in go (golang) is a small library supporting base64url encoding and decoding (collectively called coding). In addition, it is a command-line tool similar to BSD's `base64` tool.

WARNING: THIS SOFTWARE CAN'T BE ERROR-FREE, SO USE IT AT YOUR OWN RISK. I HAVE DONE MY BEST TO MAKE SURE THAT THE TOOLS BEHAVE AS EXPECTED. BUT AGAIN ... USE IT AT YOUR OWN RISK. I AM NOT GIVING ANY KIND OF WARRANTY, NEITHER EXPLICITLY NOR IMPLICITLY.

## Sample usage of library

```go
    package main

    import (
        "fmt"
        "os"

        "github.com/theovassiliou/base64url"
    )

    const testString = "A text content to be base64url encoded"
    const encodedTestString = "QSB0ZXh0IGNvbnRlbnQgdG8gYmUgYmFzZTY0dXJsIGVuY29kZWQ"

    func main() {
        // encode a []byte and get the encoded value
        output := base64url.Encode([]byte(testString))
        fmt.Println(output) // returns QSB0ZXh0IGNvbnRlbnQgdG8gYmUgYmFzZTY0dXJsIGVuY29kZWQ

        // decode a string
        decodedOutput, err := base64url.Decode(encodedTestString)

        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }

        fmt.Println(string(decodedOutput))

        // check for symmetry
        decodedOutput, _ = base64url.Decode(base64url.Encode([]byte(testString)))
        fmt.Println(string(decodedOutput))

    }
```

## Installation binaries

You can download the binaries directly from the [releases](https://github.com/theovassiliou/base64url/releases) section.  Unzip/untar the downloaded archive and copy the files to a location of your choice, e.g. `/usr/local/bin/` on *NIX or MacOS. If you install only the binaries, make sure that they are accessible from the command line. Ideally, they are accessible via `$PATH` or `%PATH%`, respectively.

## Installation From Source

base64url requires golang version 1.13 or newer, the Makefile requires GNU make.

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

There is no particular requirement beyond the fact that you should have a working go installation.

[Install Go](https://golang.org/doc/install) >=1.13

### Installing

Download base64url source by running, assuming you have git installed

```shell
cd $GOPATH/src/github.com/theovassiliou
git clone https://github.com/theovassiliou/base64url.git
```

This gets you your copy of base64url installed under
`$GOPATH/src/github.com/theovassiliou/base64url`

Run `make` from the source directory by running

```shell
cd $GOPATH/src/github.com/theovassiliou/base64url
make all
```

to compile and build the executable `base64url`

and run

```shell
make go-install
```

to install a copy of the executable into `$GOPATH/bin`

For information on how to use the executable  `base64url` consult the [README](cmd/base64url/README.md))

## Running the tests

We are using two different make targets for running tests.

```shell
make test
go test -short `go list`
ok      github.com/theovassiliou/base64url 0.3s
```

executes all short package tests, while

```shell
make test-all
go vet $(go list ./... )
go test `go list`
ok      github.com/theovassiliou/base64url 0.3s
```

executes in addition `go vet` on the package. Before committing to the code base please use `make test-all` to ensure that all tests pass.

## Contributing

Please read [CONTRIBUTING.md](https://gist.github.com/PurpleBooth/b24679402957c63ec426) for details on our code of conduct, and the process for submitting pull requests to us.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/theovassiliou/base64url/tags).

## Authors

* **Theo Vassiliou** - *Initial work* - [Theo Vassiliou](https://github.com/theovassiliou)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

## Acknowledgments

Thanks to all the people out there that produce amazing open-source software, which supported the creation of this piece of software. In particular, I wasn't only able to use libraries, etc. But also, to learn and understand golang better. In particular, I wanted to thank

* [Jaime Pillora](https://github.com/jpillora) for [jpillora/opts](https://github.com/jpillora/opts). Nice piece of work!
* [InfluxData Team](https://github.com/influxdata) for [influxdata/telegraf](https://github.com/influxdata/telegraf). Here I learned a lot for Makefile writing and release building in particular.
* [PurpleBooth](https://gist.github.com/PurpleBooth) for the well motivated [README-template](https://gist.github.com/PurpleBooth/109311bb0361f32d87a2)

***

## History

This project has been developed as I was seeking a library to support my work on identifying microservice.
