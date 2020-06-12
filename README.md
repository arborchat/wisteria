# Wisteria

[![builds.sr.ht status](https://builds.sr.ht/~whereswaldon/wisteria.svg)](https://builds.sr.ht/~whereswaldon/wisteria?)

Wisteria is a terminal chat client for the Arbor Chat Project.

## About Arbor

![arbor logo](https://git.sr.ht/~whereswaldon/forest-go/blob/master/img/arbor-logo.png)

Arbor is a chat system that makes communication clearer. It explicitly captures context that other platforms ignore, allowing you to understand the relationship between each message and every other message. It also respects its users and focuses on group collaboration.

You can get information about the Arbor project [here](https://man.sr.ht/~whereswaldon/arborchat/).

For news about the project, join our [mailing list](https://lists.sr.ht/~whereswaldon/arbor-dev)!

## Trying it out

To get started with `wisteria`, you can follow our [Getting Started with Arbor guide](https://man.sr.ht/%7Ewhereswaldon/arborchat/getting-started.md).

## FAQ

> What is this?

`wisteria` is a minimal terminal arbor client. It can receive messages directly from a relay through the [Sprout protocol](https://arbor.chat/specifications/sprout.md).

## Contributing

Want to work on `wisteria`? Here's how to do common stuff:

### Build a customized version

If you've modified your code and want to take it for a spin, you can use:

```shell
go build .
```

This will place a `wisteria` executable in the current working directory.
You can run that with:

```shell
./wisteria
```

### Running the tests

You can run all of our tests by doing:

```
go test -v -coverprofile=coverage.profile ./...
```

This will give you lots of feedback about the tests, and will also generate
a code coverage report. You can view the code coverage in your browser by
running:

```
go tool cover -html=coverage.profile
```

### Submitting a change

We accept Pull Requests three ways:

#### SourceHut

Make your own version of the code in a personal SourceHut repo. You can either
push your local clone to SourceHut or click the blue "Clone repo to your account"
button on [this page](https://git.sr.ht/~whereswaldon/wisteria) to get your own copy in SourceHut.

Once you have your own SourceHut repo for `wisteria`, click the blue "Prepare a patchset" button (in the same place that "Clone repo to your account" was).
Choose your branch and the commits within it that you'd like to submit. Once you
reach the stage with the title "Finalize the patchset", click to "Add a cover letter"
and explain what your PR is for. Feel free to also add commentary to any of the
patches.

Once you reach "Review your patchset", send the email to `~whereswaldon/arbor-dev@lists.sr.ht`. You should then be able to see your patches
and our responses to them [here on the mailing list](https://lists.sr.ht/~whereswaldon/arbor-dev).

#### GitHub

We have a [GitHub `wisteria` mirror repo](https://github.com/arborchat/wisteria). You can [submit a Pull Request](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/creating-a-pull-request) there.

#### E-mail

If you don't want to create a SourceHut account or a Github account, you can submit patches via e-mail to the mailing list managed by SourceHut.

Clone the repository from SourceHut using the anonymous https option and work as you normally would with git: make your changes, stage them, and commit them.

Once you're ready to create your PR, follow the instructions [here](https://git-send-email.io/) to actually send your patch to the mailing list at [~whereswaldon/arbor-dev@lists.sr.ht](mailto:~whereswaldon/arbor-dev@lists.sr.ht).
After you've sent your patch the process is identical to SourceHut: you can verify that your patch was received by looking at the list as well as see any responses.
