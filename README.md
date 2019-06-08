# forest-go

[![builds.sr.ht status](https://builds.sr.ht/~whereswaldon/forest-go.svg)](https://builds.sr.ht/~whereswaldon/forest-go?)
[![GoDoc](https://godoc.org/git.sr.ht/~whereswaldon/forest-go?status.svg)](https://godoc.org/git.sr.ht/~whereswaldon/forest-go)

A golang library for working with nodes in the Arbor Forest. This repo is based on the work-in-progress specification [available here](https://github.com/arborchat/protocol/blob/forest/spec/Forest.md).

## About Arbor

![arbor logo](https://git.sr.ht/~whereswaldon/forest-go/blob/master/img/arbor-logo.png)

Arbor is a chat system that makes communication clearer. It explicitly captures context that other platforms ignore, allowing you to understand the relationship between each message and every other message. It also respects its users and focuses on group collaboration.

For most of the project's components, look [here](https://github.com/arborchat).

For news about the project, join our [mailing list](https://lists.sr.ht/~whereswaldon/arbor-dev)!

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

#### Identities
Since all nodes must be signed by an Identity node, you must create one of those before you can create any others.

```sh
forest create identity --name <your-name>
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
forest show <id> | jq .
```

Substitute the base64url-encoded ID of your identity node for `<id>`. `jq` will pretty-print the JSON to make it easier to read.

#### Communities

To create a community, use:

```sh
forest create community --as <id> --name <community-name>
```

Substitute the base64url-encoded ID of your identity node for `<id>` and provide appropriate values for name and metadata.

To view your community in a human-readable format, try the following:

```sh
forest show <id> | jq .
```

Substitute the base64url-encoded ID of your community node for `<id>`. `jq` will pretty-print the JSON to make it easier to read.

#### Replies

To create a reply, use:

```sh
forest create reply --as <id> --to <parent-id> --content <your message>
```

Substitute the base64url-encoded ID of your identity node for `<id>` and the base64url-encoded ID of another reply or conversation node for `<parent-id>`. Substitute `<your message>`
for the content of your reply. Usually this will be a response to the content of the node referenced by `<parent-id>`.

To view your reply in a human-readable format, try the following:

```sh
forest show <id> | jq .
```

Substitute the base64url-encoded ID of your reply node for `<id>`. `jq` will pretty-print the JSON to make it easier to read.


## Build

Must use Go 1.11+

`go build`

## Test

`go test -v -cover`
