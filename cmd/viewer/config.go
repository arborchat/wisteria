package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// Config holds the user's runtime configuration
type Config struct {
	// a PGP key ID for the user's private key that controls their arbor identity.
	// Mutually exclusive with PGPKey
	PGPUser string
	// an unencrypted PGP private key file that controls the user's identity. Insecure,
	// and mutually exclusive with PGPUser
	PGPKey string
	// the file name of the user's arbor identity node
	IdentityName string
	// where to store log and profile data
	RuntimeDirectory string
	// The command to launch an editor for composing new messages
	EditorCmd []string
}

// NewConfig creates a config that is prepopulated with a runtime directory and an editor command that
// will work on many Linux systems
func NewConfig() *Config {
	dir, err := ioutil.TempDir("", "arbor")
	if err != nil {
		log.Println("Failed to create temporary runtime directory, falling back to os-global temp dir")
		dir = os.TempDir()
	}
	return &Config{
		RuntimeDirectory: dir,
		EditorCmd:        []string{"xterm", "-e", os.ExpandEnv("$EDITOR"), "{}"},
	}
}

// Validate errors if the configuration is invalid
func (c *Config) Validate() error {
	switch {
	case c.PGPUser != "" && c.PGPKey != "":
		return fmt.Errorf("PGPUser and PGPKey cannot both be set")
	case c.PGPUser == "" && c.PGPKey == "":
		return fmt.Errorf("One of PGPUser and PGPKey must be set")
	case c.IdentityName == "":
		return fmt.Errorf("IdentityName must be set")
	case len(c.EditorCmd) < 2:
		return fmt.Errorf("Editor Command %v is impossibly short", c.EditorCmd)
	}
	return nil
}

// EditFile returns an exec.Cmd that will open the provided filename, edit it, and block until the
// edit is completed.
func (c *Config) EditFile(filename string) *exec.Cmd {
	out := make([]string, 0, len(c.EditorCmd))
	for _, part := range c.EditorCmd {
		if part == "{}" {
			out = append(out, filename)
		} else {
			out = append(out, part)
		}
	}
	return exec.Command(out[0], out[1:]...)
}

// Builder creates a forest.Builder based on the configuration. This allows the client
// to create nodes on this user's behalf.
func (c *Config) Builder() (*forest.Builder, error) {
	var (
		signer forest.Signer
		err    error
	)
	if c.PGPUser != "" {
		signer, err = forest.NewGPGSigner(c.PGPUser)
	} else if c.PGPKey != "" {
		keyfile, _ := os.Open(c.PGPKey)
		defer keyfile.Close()
		entity, _ := openpgp.ReadEntity(packet.NewReader(keyfile))
		signer, err = forest.NewNativeSigner(entity)
	}
	if err != nil {
		log.Fatal(err)
	}
	idBytes, err := ioutil.ReadFile(c.IdentityName)
	if err != nil {
		log.Fatal(err)
	}
	identity, err := forest.UnmarshalIdentity(idBytes)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	return forest.As(identity, signer), nil
}

// StdoutPrompter asks the user to make choices in an interactive text prompt
type StdoutPrompter struct {
	Out io.Writer
	In  io.Reader
}

// Choose asks the user to choose from among a list of options. The formatter
// function is used to display each option to the user
func (s *StdoutPrompter) Choose(prompt string, slice []interface{}, formatter func(element interface{}) string) (choice interface{}, err error) {
	in := bufio.NewReader(s.In)
	success := false
	attempts := 0
	index := 0
	const maxAttempts = 5
	for !success && attempts < maxAttempts {
		fmt.Fprintln(s.Out)
		attempts++
		fmt.Fprintln(s.Out, prompt)
		for i, v := range slice {
			fmt.Fprintf(s.Out, "\t%d) %s\n", i, formatter(v))
		}
		fmt.Print("Your choice: ")
		str, err := in.ReadString("\n"[0])
		if err != nil {
			fmt.Fprintf(s.Out, "Error reading input: %v", err)
			continue
		}
		index, err = strconv.Atoi(strings.TrimSuffix(str, "\n"))
		if err != nil {
			fmt.Fprintf(s.Out, "Input must be a number: %v", err)
			continue
		}
		if index >= len(slice) || index < 0 {
			fmt.Fprintf(s.Out, "Index %d is out of range", index)
			continue
		}
		success = true
	}
	if !success {
		return nil, fmt.Errorf("max input attempts exceeded")
	}
	return slice[index], nil
}

func KeyFrom(id *forest.Identity) (*openpgp.Entity, error) {
	buf := bytes.NewBuffer(id.PublicKey.Blob)
	entity, err := openpgp.ReadEntity(packet.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("Error reading public key from %v: %v", id.ID(), err)
	}
	return entity, nil
}

// RunWizard populates the config by asking the user for information and
// inferring from the runtime environment
func RunWizard(cwd string, config *Config) error {
	identities := []interface{}{}
	for _, node := range NodesFromDir(cwd) {
		if id, ok := node.(*forest.Identity); ok {
			identities = append(identities, id)
		}
	}
	// ensure that we have a typed nil to represent a the choice to create a new
	// identity
	var makeNew *forest.Identity = nil
	identities = append(identities, makeNew)
	prompter := &StdoutPrompter{In: os.Stdin, Out: os.Stdout}
	choiceInterface, err := prompter.Choose("Please choose an identity:", identities, func(i interface{}) string {
		id := i.(*forest.Identity)
		if id == nil {
			return "create a new identity (unimplemented)"
		}
		idString, err := id.ID().MarshalString()
		if err != nil {
			return fmt.Sprintf("Error formatting ID() into string: %v", err)
		}
		return fmt.Sprintf("%-16s %60s", string(id.Name.Blob), idString)
	})
	if err != nil {
		return fmt.Errorf("Error reading user response: %v", err)
	}
	choice := choiceInterface.(*forest.Identity)
	if choice == nil {
		fmt.Println("Creating a new identity is not yet supported")
		return fmt.Errorf("Creating a new identity is not yet supported")
	}

	name, err := choice.ID().MarshalString()
	if err != nil {
		return fmt.Errorf("Error marshalling identity string: %v", err)
	}
	config.IdentityName = name
	key, err := KeyFrom(choice)
	if err != nil {
		return fmt.Errorf("Error extracting key: %v", err)
	}
	pgpIds := []string{}
	for keyID := range key.Identities {
		pgpIds = append(pgpIds, keyID)
	}
	config.PGPUser = pgpIds[0]
	return nil
}
