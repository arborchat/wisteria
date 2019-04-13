package main

import (
	"flag"
	"fmt"
	"os"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"golang.org/x/crypto/openpgp"
)

func main() {
	var (
		pgpName, pgpComment, pgpEmail, username string
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s: Write a binary identity node to stdout\nUsage:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&pgpName, "pgp-name", "Forest Demo", "The name to include in the OpenPGP Key's metadata")
	flag.StringVar(&pgpComment, "pgp-comment", "Example OpenPGP Key", "The name to include in the OpenPGP Key's metadata")
	flag.StringVar(&pgpName, "pgp-email", "example@arbor.chat", "The name to include in the OpenPGP Key's metadata")
	flag.StringVar(&username, "arbor-username", "Example User", "The Arbor username for this user")
	flag.Parse()

	privkey, err := openpgp.NewEntity(pgpName, pgpComment, pgpEmail, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	qUsername, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte(username))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	qMetadata, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte(""))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	builder := forest.IdentityBuilder{}
	identity, err := builder.New(privkey, qUsername, qMetadata)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	b, err := identity.MarshalBinary()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = os.Stdout.Write(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
