# What is this?

`viewer` (final name TBD) is a minimal terminal arbor client. It is capable of interactively rendering messages stored on disk into a scrollable tree and creating new reply nodes. It also detects new files on disk and loads their contents automatically (if they are arbor nodes).

> So if it only renders what's on disk, how can I talk to someone?

Well, the key is that it live-loads any new arbor nodes that appear in its current working directory. This means that you can establish a shared folder using any number of file synchronization tools and that folder will replicate its contents between you and whatever peers you share it with. When you write a new node into that folder, it will replicate across the network to your peer, and their client will discover it and load it into their instance of `viewer`. Depending on the replication, there may be some lag.

## Installing viewer

You need Go 1.12+ to install this. You may be able to get this with your package manager, but you can always download it from [here](https://golang.org/dl/).

Clone this repo, check out this branch (currently `viewer`), and run `go install ./...`. Make sure `~/go/bin` (or `$GOPATH/bin` if you have a custom `$GOPATH`) is in your path.

## Running viewer

Running `viewer` currently requires some prep work (we'll eliminate most of these steps soon).

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

`viewer` currently can't begin new conversations (though it's coming soon). In order to have a conversation to talk in, we can just make one from our shell. If you're joining people who are already talking, this is likely unnecessary.

```bash
forest create reply --gpguser <email> --to <community_file_name> --as <identity_file_name> --content "<your message>"
```

- Replace `<email>` with the email that you associated with your GPG key.
- Replace `<identity_file_name>` with the output of the `forest create identity ...` command.
- Replace `<community_file_name>` output of the `forest create community ...` command.
- Replace `<your message` with the message that you want to start out the conversation. Make sure to quote it!

You won't need the output of this command later.

### Figure out your editor command

`viewer` uses your favorite editor program to write new messages. This means that you need to tell `viewer` how to launch your favorite editor.

Specifically, you need to figure out how to launch your favorite editor so that it:

- Does not try to take over the same terminal that `viewer` is using (this means that users of terminal editors will need to open a new terminal emulator or terminal multiplexer pane). This restriction will be lifted in the future.
- Opens a file provided in its command line arguments.
- Blocks the shell that launched it until you save and quit. **IMPORTANT:** If the shell does not block waiting for the command to finish, you will send empty arbor messages.

Here are some examples:

```bash
# Launch a new gnome-terminal with your favorite editor and open the file foo
gnome-terminal --wait -- $EDITOR foo

# Same using xterm instead of gnome-terminal
xterm -e $EDTIOR foo

# Use GNOME's GUI text editing application
gedit foo
```

### Start `viewer`

Okay, so now we run `viewer` itself in the directory where you'd like to store your history. This directory should already contain your arbor identity and at least one community and reply (either ones you created or pre-existing ones).

```bash
viewer --gpguser <email> --identity <identity_file_name> [<editor_command> [<editor_command_options>]+]
```

- Replace `<email>` with the email that you associated with your GPG key.
- Replace `<identity_file_name>` with the output of the `forest create identity ...` command.
- After both of the flags, type your Editor Command. Use the string `'{}'` where the name of a file to edit should go.

An example of running `viewer` looks like:

```bash
viewer --gpguser christopher.waldon.dev@gmail.com -identity SHA512_B32__j1Yj6FOUefHJDI30C-fg5pjW2XkXoQ09BWbMox597pE gnome-terminal --wait -- kak '{}'
```
