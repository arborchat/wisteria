# forest-go

[![builds.sr.ht status](https://builds.sr.ht/~whereswaldon/forest-go.svg)](https://builds.sr.ht/~whereswaldon/forest-go?)
[![GoDoc](https://godoc.org/git.sr.ht/~whereswaldon/forest-go?status.svg)](https://godoc.org/git.sr.ht/~whereswaldon/forest-go)

A golang library for working with nodes in the Arbor Forest. The `cmd/` subdirectory contains utilities for creating and examining Forest Nodes.

This repo is based on the work-in-progress specification [available here](https://github.com/arborchat/protocol/blob/forest/spec/Forest.md).

## Command Line Interface

This project includes both a Go library for manipulating nodes in the Arbor Forest and a CLI for doing so.

### Installing the CLI

The CLI is in `./cmd/forest/`, and you can install it with:

```sh
go get -u git.sr.ht/~whereswaldon/forest-go/cmd/forest
```

### Using the CLI

Right now, the CLI works with files in its current working directory, though this will change in the future.
For the meantime, create a directory to play around in:

```sh
mkdir arbor-forest
cd arbor-forest
```

Since all nodes must be signed by an Identity node, you must create one of those before you can create any others.

```sh
forest identity create --name <your-name> --metadata <anything-you-want>
```

This will print the base64url-encoded ID of your identity node, which will be stored in a file by that name in your
current working directory.

> **A note about OpenPGP Keys**
> 
> Your identity will also use an OpenPGP Private Key. In the above configuration, the CLI will create a new one for you and store it
> in `./arbor.privkey`. This private key is not encrypted (has no passphrase), and should not be used for anything of importance.
> You can supply your own OpenPGP private key for an identity, but `forest` does not currently support private keys with passphrases
> (a major drawback that will be addressed soon). You can use the `--key` flag to supply a key.

To view your identity in a human-readable format, try the following (install `jq` if you don't have it, it's really handy):

```sh
forest identity show <id> | jq .
```

Substitute the base64url-encoded ID of your identity node for `<id>`. `jq` will pretty-print the JSON to make it easier to read.

To create a community, use:

```sh
forest community create --as <id> --name <community-name> --metadata <anything-you-want>
```

Substitute the base64url-encoded ID of your identity node for `<id>` and provide appropriate values for name and metadata.

To view your community in a human-readable format, try the following:

```sh
forest community show <id> | jq .
```

Substitute the base64url-encoded ID of your community node for `<id>`. `jq` will pretty-print the JSON to make it easier to read.

## Build

Must use Go 1.11+

`go build`

## Test

`go test -v -cover`
