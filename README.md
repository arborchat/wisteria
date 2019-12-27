# Wisteria

[![builds.sr.ht status](https://builds.sr.ht/~whereswaldon/wisteria.svg)](https://builds.sr.ht/~whereswaldon/wisteria?)

Wisteria is a terminal chat client for the Arbor Chat Project.

## About Arbor

![arbor logo](https://git.sr.ht/~whereswaldon/forest-go/blob/master/img/arbor-logo.png)

Arbor is a chat system that makes communication clearer. It explicitly captures context that other platforms ignore, allowing you to understand the relationship between each message and every other message. It also respects its users and focuses on group collaboration.

You can get information about the Arbor project [here](https://man.sr.ht/~whereswaldon/arborchat/).

For news about the project, join our [mailing list](https://lists.sr.ht/~whereswaldon/arbor-dev)!

# What is this?

`wisteria` is a minimal terminal arbor client. It is capable of interactively rendering messages stored on disk into a scrollable tree and creating new reply nodes. It also detects new files on disk and loads their contents automatically (if they are arbor nodes).

> So if it only renders what's on disk, how can I talk to someone?

Well, the key is that it live-loads any new arbor nodes that appear in its current working directory. This means that you can exchange messages with anything from a shared folder (syncthing, dropbox, google drive, etc...) to our relay infrastructure. We recommend that you start with relays, as they have the least latency.

# Trying it out

`wisteria` currently needs you to have a running [arbor `relay`]() in order to exchange messages with other users. To set everything up together, you can follow our [Getting Started with Arbor guide](https://man.sr.ht/%7Ewhereswaldon/arborchat/getting-started.md).
