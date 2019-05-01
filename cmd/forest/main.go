package main

import (
	"encoding"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

const (
	usageError = 1

	commandIdentity = "identity"

	subcommandCreate = "create"
)

func usageAbort() {
	flag.PrintDefaults()
	os.Exit(usageError)
}

func main() {
	if len(os.Args) < 2 {
		usageAbort()
	}

	var cmdHandler handler
	switch os.Args[1] {
	case commandIdentity:
		cmdHandler = identity
	default:
		usageAbort()
	}
	if err := cmdHandler(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

type handler func(args []string) error

func identity(args []string) error {
	if len(args) < 1 {
		usageAbort()
	}
	var cmdHandler handler
	switch args[0] {
	case subcommandCreate:
		cmdHandler = createIdentity
	default:
		usageAbort()
	}
	if err := cmdHandler(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return nil
}

func createIdentity(args []string) error {
	var (
		name, metadata, keyfile string
	)
	flags := flag.NewFlagSet("create identity", flag.ContinueOnError)
	flags.StringVar(&name, "name", "forest", "username for the identity node")
	flags.StringVar(&metadata, "metadata", "forest", "metadata for the identity node")
	flags.StringVar(&keyfile, "key", "arbor.privkey", "the openpgp private key for the identity node")
	if err := flags.Parse(args); err != nil {
		flags.PrintDefaults()
		return err
	}
	qName, err := fields.NewQualifiedContent(fields.ContentTypeUTF8String, []byte(name))
	if err != nil {
		return err
	}
	qMeta, err := fields.NewQualifiedContent(fields.ContentTypeUTF8String, []byte(metadata))
	if err != nil {
		return err
	}
	privkey, err := getPrivateKey(keyfile, &PGPKeyConfig{
		Name:    "Arbor identity key",
		Comment: "Automatically generated",
		Email:   "none@arbor.chat",
	})
	if err != nil {
		return err
	}
	identity, err := forest.NewIdentity(privkey, qName, qMeta)
	if err != nil {
		return err
	}

	fname, err := filename(identity.ID())
	if err != nil {
		return err
	}

	outfile, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer outfile.Close()

	if err := save(outfile, identity); err != nil {
		return err
	}

	fmt.Println(fname)

	return nil
}

func filename(desc *fields.QualifiedHash) (string, error) {
	b, err := desc.MarshalBinary()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func save(w io.Writer, node encoding.BinaryMarshaler) error {
	b, err := node.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err

}

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
