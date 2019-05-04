package main

import (
	"encoding"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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
	subcommandShow   = "show"
)

func main() {
	flag.Usage = func() {
		flag.PrintDefaults()
		os.Exit(usageError)
	}
	if len(os.Args) < 2 {
		flag.Usage()
	}

	var cmdHandler handler
	switch os.Args[1] {
	case commandIdentity:
		cmdHandler = identity
	default:
		flag.Usage()
	}
	if err := cmdHandler(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

type handler func(args []string) error

func identity(args []string) error {
	flags := flag.NewFlagSet(commandIdentity, flag.ExitOnError)
	usage := func() {
		flags.PrintDefaults()
		os.Exit(usageError)
	}
	err := flags.Parse(args)
	if err != nil {
		return err
	}
	if len(flags.Args()) < 1 {
		usage()
	}
	var cmdHandler handler
	switch flags.Arg(0) {
	case subcommandCreate:
		cmdHandler = createIdentity
	case subcommandShow:
		cmdHandler = showIdentity
	default:
		usage()
	}
	if err := cmdHandler(flags.Args()[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return nil
}

func createIdentity(args []string) error {
	var (
		name, metadata, keyfile string
	)
	flags := flag.NewFlagSet(commandIdentity+" "+subcommandCreate, flag.ContinueOnError)
	flags.StringVar(&name, "name", "forest", "username for the identity node")
	flags.StringVar(&metadata, "metadata", "forest", "metadata for the identity node")
	flags.StringVar(&keyfile, "key", "arbor.privkey", "the openpgp private key for the identity node")
	usage := func() {
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		usage()
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

func showIdentity(args []string) error {
	flags := flag.NewFlagSet(commandIdentity+" "+subcommandShow, flag.ContinueOnError)
	usage := func() {
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		usage()
		return err
	}
	if len(flags.Args()) < 1 {
		return fmt.Errorf("missing required argument [node id]")
	}
	idFile, err := os.Open(args[0])
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(idFile)
	if err != nil {
		return err
	}
	i, err := forest.UnmarshalIdentity(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	text, err := json.Marshal(i)
	if err != nil {
		return err
	}
	if _, err := os.Stdout.Write(text); err != nil {
		return err
	}
	return nil
}

func filename(desc *fields.QualifiedHash) (string, error) {
	b, err := desc.MarshalBinary()
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
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
