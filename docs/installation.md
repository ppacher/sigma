# Installation

## Installation from Source

In order to install Sigma from source a working [Golang](https://golang.org) environment is required. 

```bash
go get github.com/homebot/sigma
go get github.com/homebot/sigma/cmd/sigma...

go install github.com/homebot/sigma/cmd/simga
```

Assuming that `$GOPATH/bin` (aka `$GOBIN`) is in your `$PATH` you should be able to access the sigma binary:

```
sigma --help
```

### Building protocol buffer and gRPC

The steps above downloads and uses the pre-built go bindings for the protobuf messages located at [homebot/protobuf](https://github.com/homebot/protobuf). If you want to re-build protobuf messages, clone the repository into `$GOPATH` and use `make`:

```bash
$ git clone https://github.com/homebot/protobuf $GOPATH/src/github.com/homebot/protobuf
$ cd $GOPATH/src/github.com/homebot/protobuf

$ make
```