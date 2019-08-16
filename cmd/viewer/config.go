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
	Identity *forest.Identity
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
	case c.Identity == nil:
		return fmt.Errorf("Identity must be set")
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
	return forest.As(c.Identity, signer), nil
}

// Unixify ensures that a string contains only unix-style newlines, converting
// windows-style ones as necessary
func Unixify(in string) string {
	return strings.ReplaceAll(in, "\r\n", "\n")
}

// Prompter can display text to the user and can ask them to make simple choices.
type Prompter interface {
	Choose(prompt string, slice []interface{}, formatter func(element interface{}) string) (choice interface{}, err error)
	PromptLine(prompt string) (input string, err error)
	Display(message string) error
}

// StdoutPrompter asks the user to make choices in an interactive text prompt
type StdoutPrompter struct {
	Out io.Writer
	In  io.Reader
}

// Choose asks the user to choose from among a list of options. The formatter
// function is used to display each option to the user
func (s *StdoutPrompter) Choose(prompt string, slice []interface{}, formatter func(element interface{}) string) (choice interface{}, err error) {
	if len(slice) < 1 {
		return nil, fmt.Errorf("Cannot choose from empty option list")
	}
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
		index, err = strconv.Atoi(strings.ReplaceAll(Unixify(str), "\n", ""))
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

// PromptLine asks the user for a single line of free-form input text
func (s *StdoutPrompter) PromptLine(prompt string) (input string, err error) {
	in := bufio.NewReader(s.In)
	success := false
	attempts := 0
	const maxAttempts = 5
	for !success && attempts < maxAttempts {
		fmt.Fprintln(s.Out)
		attempts++
		fmt.Fprintln(s.Out, prompt)
		input, err = in.ReadString("\n"[0])
		if err != nil {
			fmt.Fprintf(s.Out, "Error reading input: %v", err)
			continue
		}
		input = strings.TrimSpace(input)
		if len(input) < 1 {
			fmt.Fprintf(s.Out, "Cannot use only whitespace")
			continue
		}
		success = true
	}
	if !success {
		return "", fmt.Errorf("max input attempts exceeded")
	}
	return input, nil
}

// Display shows a message to the user
func (s *StdoutPrompter) Display(message string) error {
	_, err := fmt.Fprintln(s.Out, message)
	return err
}

func KeyFrom(id *forest.Identity) (*openpgp.Entity, error) {
	buf := bytes.NewBuffer(id.PublicKey.Blob)
	entity, err := openpgp.ReadEntity(packet.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("Error reading public key from %v: %v", id.ID(), err)
	}
	return entity, nil
}

func GetSecretKeys() ([]string, error) {
	gpgCommand, err := forest.FindGPG()
	if err != nil {
		return nil, fmt.Errorf("Failed to find gpg installation: %v", err)
	}
	cmd := exec.Command(gpgCommand, "--list-secret-keys", "--with-colons")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to create gpg stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Failed starting to list gpg secret keys: %v", err)
	}
	b, err := ioutil.ReadAll(out)
	if err != nil {
		return nil, fmt.Errorf("Failed reading gpg stdout: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("Failed listing gpg secret keys: %v", err)
	}
	lines := strings.Split(string(b), "\n")
	ids := []string{}
	const commentPosition = 9 // the field number of the user info comment
	for _, line := range lines {
		if strings.HasPrefix(line, "uid") {
			ids = append(ids, strings.Split(line, ":")[commentPosition])
		}
	}
	return ids, nil
}

type Wizard struct {
	Prompter
	*Config
}

// ConfigureNewIdentity creates a completely new identity using an existing GPG key
func (w *Wizard) ConfigureNewIdentity() error {
	secKeys, err := GetSecretKeys()
	if err != nil {
		return fmt.Errorf("Failed to list available secret keys: %v", err)
	}
	asInterface := make([]interface{}, len(secKeys))
	for i := range secKeys {
		asInterface[i] = secKeys[i]
	}
	const createNewOption = "Create a new key"
	asInterface = append(asInterface, createNewOption)
	secKey, err := w.Choose("Choose a gpg private key for this identity:", asInterface, func(i interface{}) string {
		return i.(string)
	})
	if secKey.(string) == createNewOption {
		fmt.Printf("\nTo create a new key, run:\n\ngpg2 --generate-key\n\nRe-run %v when you've done that.\n", os.Args[0])
		return fmt.Errorf("Closing so that you can generate a key")
	}
	signer, err := forest.NewGPGSigner(secKey.(string))
	if err != nil {
		return fmt.Errorf("Unable to construct a signer from gpg key for %s: %v", secKey, err)
	}
	username, err := w.PromptLine("Enter a username:")
	if err != nil {
		return fmt.Errorf("Failed to get username: %v", err)
	}
	identity, err := forest.NewIdentity(signer, username, "")
	if err != nil {
		return fmt.Errorf("Failed to create identity: %v", err)
	}
	name, err := identity.ID().MarshalString()
	if err != nil {
		return fmt.Errorf("Error marshalling identity string: %v", err)
	}
	if err := saveAs(name, identity); err != nil {
		return fmt.Errorf("Error saving new identity %s: %v", name, err)
	}
	w.Identity = identity
	return nil
}

// ConfigureIdentity sets up an identity in the Wizard's config. It creates a new one
// if the user requests it.
func (w *Wizard) ConfigureIdentity(cwd string) error {
	identities := []interface{}{}
	for _, node := range NodesFromDir(cwd) {
		if id, ok := node.(*forest.Identity); ok {
			identities = append(identities, id)
		}
	}
	// ensure that we have a typed nil to represent a the choice to create a new identity
	var makeNew *forest.Identity = nil
	identities = append(identities, makeNew)
	choiceInterface, err := w.Choose("Please choose an identity:", identities, func(i interface{}) string {
		id := i.(*forest.Identity)
		if id == nil {
			return "create a new identity"
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
	if choice != nil {
		w.Identity = choice
		return nil
	}

	return w.ConfigureNewIdentity()
}

// ConfigureEditor walks the user through choosing an editor command for their client.
func (w *Wizard) ConfigureEditor() error {
	editors := []interface{}{}
	for _, ed := range FindEditors() {
		editors = append(editors, ed)
	}
	choiceInterface, err := w.Choose("Please choose a command to edit messages with:", editors, func(i interface{}) string {
		return strings.Join(KnownEditorCommands[i.(string)], " ")
	})
	if err != nil {
		return fmt.Errorf("Error reading user response: %v", err)
	}

	w.EditorCmd = KnownEditorCommands[choiceInterface.(string)]
	return nil
}

// Run populates the config by asking the user for information and
// inferring from the runtime environment
func (w *Wizard) Run(cwd string) error {
	err := w.ConfigureIdentity(cwd)
	if err != nil {
		return fmt.Errorf("Error configuring user identity: %v", err)
	}
	key, err := KeyFrom(w.Identity)
	if err != nil {
		return fmt.Errorf("Error extracting key: %v", err)
	}
	pgpIds := []string{}
	for keyID := range key.Identities {
		pgpIds = append(pgpIds, keyID)
	}
	w.PGPUser = pgpIds[0]
	if err := w.ConfigureEditor(); err != nil {
		return fmt.Errorf("Error configuring editor command: %v", err)
	}
	return nil
}

func FindEditors() []string {
	out := []string{}
	for term := range KnownEditorCommands {
		if _, err := exec.LookPath(term); err == nil {
			out = append(out, term)
		}
	}
	return out
}

func ExpandAll(in []string) []string {
	for i, s := range in {
		in[i] = os.Expand(s, func(in string) string {
			if val, ok := os.LookupEnv(in); !ok {
				return fmt.Sprintf("[set $%s to use]", in)
			} else {
				return val
			}
		})
	}
	return in
}

var KnownEditorCommands = map[string][]string{
	"xterm":          ExpandAll([]string{"xterm", "-e", "$EDITOR", "{}"}),
	"gnome-terminal": ExpandAll([]string{"gnome-terminal", "--wait", "--", "$EDITOR", "{}"}),
	"gedit":          {"gedit", "{}"},
	"notepad":        {"notepad", "{}"},
}
