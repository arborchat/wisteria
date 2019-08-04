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

// RunWizard populates the config by asking the user for information and
// inferring from the runtime environment
func RunWizard(cwd string, config *Config) error {
	in := bufio.NewReader(os.Stdin)
	prompt := func(out string) (int, error) {
		fmt.Print(out)
		s, err := in.ReadString("\n"[0])
		if err != nil {
			return 0, err
		}
		index, err := strconv.Atoi(strings.TrimSuffix(s, "\n"))
		if err != nil {
			return 0, fmt.Errorf("Error decoding user response to integer: %v", err)
		}
		return index, nil
	}
	identities := []*forest.Identity{}
	for _, node := range NodesFromDir(cwd) {
		if id, ok := node.(*forest.Identity); ok {
			identities = append(identities, id)
		}
	}
	keys := make([]*openpgp.Entity, len(identities))
	for i, id := range identities {
		buf := bytes.NewBuffer(id.PublicKey.Blob)
		entity, err := openpgp.ReadEntity(packet.NewReader(buf))
		if err != nil {
			return fmt.Errorf("Error reading public key from %v: %v", id.ID(), err)
		}
		keys[i] = entity
	}
	fmt.Println("Please choose an identity:")
	for i, id := range identities {
		idString, err := id.ID().MarshalString()
		if err != nil {
			return fmt.Errorf("Error formatting ID() into string: %v", err)
		}
		fmt.Printf("%4d) %16s %60s\n", i, string(id.Name.Blob), idString)
	}
	index, err := prompt("Your choice: ")
	if err != nil {
		return fmt.Errorf("Error reading user response: %v", err)
	}
	name, err := identities[index].ID().MarshalString()
	if err != nil {
		return fmt.Errorf("Error marshalling identity string: %v", err)
	}
	config.IdentityName = name
	pgpIds := []string{}
	for key := range keys[index].Identities {
		pgpIds = append(pgpIds, key)
	}
	config.PGPUser = pgpIds[0]
	return nil
}
