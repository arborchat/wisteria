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

Well, the key is that it live-loads any new arbor nodes that appear in its current working directory. This means that you can establish a shared folder using any number of file synchronization tools and that folder will replicate its contents between you and whatever peers you share it with. When you write a new node into that folder, it will replicate across the network to your peer, and their client will discover it and load it into their instance of `wisteria`. Depending on the replication, there may be some lag.

## Installing wisteria

You need Go 1.12+ to install this. You may be able to get this with your package manager, but you can always download it from [here](https://golang.org/dl/).

Clone this repo, check out this branch (currently `wisteria`), and run `go install ./...`. Make sure `~/go/bin` (or `$GOPATH/bin` if you have a custom `$GOPATH`) is in your path.

## Running wisteria

Running `wisteria` currently requires some prep work (we'll eliminate most of these steps soon).

### Install the forest CLI

If you have a recent version of Go, you can get this with:

```
go get git.sr.ht/~whereswaldon/forest-go/cmd/forest@latest
```

Otherwise clone that repo, `cd cmd/forest` and `go install`.

### Make a GPG Key

First, you need a real GPG key. If you already have one, great! Note the email address that you associated with it (`gpg -k` will help you find that). If you don't have one, you can generate one by following the directions [here](https://wiki.archlinux.org/index.php/GnuPG#Create_a_key_pair).

Once you've got a key, try using it:

```bash
echo test | gpg2 --sign
```

If this creates a desktop popup asking for your private key passphrase, proceed. If this creates an in-terminal dialogue asking for your passphrase, you'll need to [adjust your GPG pinentry settings](https://wiki.archlinux.org/index.php/GnuPG#pinentry) so that it uses a popup window instead. This limitation will be removed in the future.

### Choose a storage location

Now you need a folder to store your Arbor history inside of. Perhaps `~/Documents/ArborHistory/`? Or the windows equivalent? Whatever you choose, get a shell with your current working directory there.

### Create an Arbor Identity

Now we need to set up your identity:

```bash
forest create identity --gpguser <email> --name <username>
```

- Replace `<email>` with the email that you associated with your GPG key.
- Replace `<username>` with the username that you want to use within Arbor.
 
Note the output of this command, as it's the name of your identity on disk. You'll need it soon.

### Create a community [skip if joining existing community]

If you are setting up in a completely empty directory, you'll need to start a community as well. If you're joining some friends who are already talking in arbor, you can probably skip this step.

```bash
forest create community --gpguser <email> --as <identity_file_name> --name <community_name>
```

- Replace `<email>` with the email that you associated with your GPG key.
- Replace `<identity_file_name>` with the output of the `forest create identity ...` command.
- Replace `<community_name>` with the name that you'd like to give to your community.
 
Note the output of this command, as it is the name of your community on disk. You'll need it later.

### Start a conversation [skip if joining existing community]

`wisteria` currently can't begin new conversations (though it's coming soon). In order to have a conversation to talk in, we can just make one from our shell. If you're joining people who are already talking, this is likely unnecessary.

```bash
forest create reply --gpguser <email> --to <community_file_name> --as <identity_file_name> --content "<your message>"
```

- Replace `<email>` with the email that you associated with your GPG key.
- Replace `<identity_file_name>` with the output of the `forest create identity ...` command.
- Replace `<community_file_name>` output of the `forest create community ...` command.
- Replace `<your message` with the message that you want to start out the conversation. Make sure to quote it!

You won't need the output of this command later.

### Start `wisteria`

Okay, so now we run `wisteria` itself in the directory where you'd like to store your history. This directory should already contain your arbor identity and at least one community and reply (either ones you created or pre-existing ones).

```bash
wisteria
```

You should be asked to select a user account (use one that is you, you won't be able to send messages otherwise). You'll also be asked to select and editor. Choose your favorite option of the list. This program will be launched to compose new replies. When you're done writing a reply, save the file and quit the editor to send the message.
