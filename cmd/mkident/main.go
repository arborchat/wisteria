package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

func readKey(in io.Reader) (*openpgp.Entity, error) {
	return openpgp.ReadEntity(packet.NewReader(in))
}

type PGPKeyConfig struct {
	Name    string
	Comment string
	Email   string
}

// getPrivateKey gets a private key for creating the identity based on the value
// of filename. If filename is:
// "-" => read a private key from stdin, do not write private key to a file
// existing file => read key from file, do not write private key to a file
// nonexistent file => create new private key, write to filename
//
// the value of config is only used when creating a new key
func getPrivateKey(filename string, config *PGPKeyConfig) (*openpgp.Entity, error) {
	var privkey *openpgp.Entity
	var err error
	if filename == "-" {
		// if stdin, try to read key
		return readKey(os.Stdin)

	}
	// check if privkeyfile exists
	keyOutFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0400)
	if err != nil {
		// keyfile may exist, use key from it
		keyOutFile, err := os.Open(filename)
		if err != nil {
			// keyfile doesn't exist or we can't open it
			return nil, err
		}
		return readKey(keyOutFile)
	}
	// keyfile did not exist, create new key and write it there
	privkey, err = openpgp.NewEntity(config.Name, config.Comment, config.Email, nil)
	if err != nil {
		return nil, err
	}

	if err := privkey.SerializePrivate(keyOutFile, nil); err != nil {
		return nil, err
	}
	return privkey, nil

}

func main() {
	var (
		pgpName, pgpComment, pgpEmail, username, privkeyOutFile string
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s: Write a binary identity node to stdout\nUsage:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&pgpName, "pgp-name", "Forest Demo", "The name to include in the OpenPGP Key's metadata")
	flag.StringVar(&pgpComment, "pgp-comment", "Example OpenPGP Key", "The name to include in the OpenPGP Key's metadata")
	flag.StringVar(&pgpName, "pgp-email", "example@arbor.chat", "The name to include in the OpenPGP Key's metadata")
	flag.StringVar(&username, "arbor-username", "Example User", "The Arbor username for this user")
	flag.StringVar(&privkeyOutFile, "privkey-file", "arbor.privkey", "The private key for this identity. Use '-' to read the privkey from stdin (this will not write it to an output file). Specify a nonexistent file if you want to create a new key")
	flag.Parse()

	privkey, err := getPrivateKey(privkeyOutFile, &PGPKeyConfig{
		Name:    pgpName,
		Comment: pgpComment,
		Email:   pgpEmail,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	qUsername, err := fields.NewQualifiedContent(fields.ContentTypeUTF8String, []byte(username))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	qMetadata, err := fields.NewQualifiedContent(fields.ContentTypeUTF8String, []byte(""))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	identity, err := forest.NewIdentity(privkey, qUsername, qMetadata)
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
