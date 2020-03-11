package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// Config holds the user's runtime configuration
type Config struct {
	// a PGP key ID for the user's private key that controls their arbor identity.
	PGPUser string
	// the file name of the user's arbor identity node
	IdentityID string
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

// StartLogging configures logging to a file chosen based on the Config. If
// any io.Writers are provided, they will all receive logs in addition to the
// configured log file.
func (c *Config) StartLogging(additionalLogSinks ...io.Writer) error {
	logPath := filepath.Join(c.RuntimeDirectory, "viewer.log")
	log.Println("Logging to", logPath)
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return fmt.Errorf("Failed to open log file %s: %w", logPath, err)
	}
	writers := append([]io.Writer{logFile}, additionalLogSinks...)
	log.SetOutput(io.MultiWriter(writers...))
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	return nil
}

// DefaultConfigFilePath returns the path at which configuration should be stored
// by default on the current OS and for the current user.
func DefaultConfigFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed looking up configuration dir: %w", err)
	}
	const wisteriaConfigDirName = "wisteria"
	const wisteriaConfigFileNameJSON = "wisteria-config.json"
	configFile := filepath.Join(configDir, wisteriaConfigDirName, wisteriaConfigFileNameJSON)
	return configFile, nil
}

func DefaultGrovePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed finding user home directory: %w", err)
	}
	const wisteriaHistoryParentDir = "Documents"
	const wisteriaHistoryDirName = "ArborHistory"
	wisteriaHistoryPath := filepath.Join(homeDir, wisteriaHistoryParentDir, wisteriaHistoryDirName)
	return wisteriaHistoryPath, nil
}

func (c *Config) LoadFromPath(configPath string) error {
	configFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("unable to open config file: %w", err)
	}
	defer configFile.Close()
	if err := c.LoadFrom(configFile); err != nil {
		return fmt.Errorf("unable to load config file: %w", err)
	}
	return nil
}

// LoadFrom loads the configuration from the given ReadCloser and closes it. It will error if
// it fails to read, parse, or validate the configuration that it reads.
func (c *Config) LoadFrom(configFile io.Reader) error {
	decoder := json.NewDecoder(configFile)
	if err := decoder.Decode(c); err != nil {
		return fmt.Errorf("failed decoding config file: %w", err)
	}
	if err := c.Validate(); err != nil {
		return fmt.Errorf("failed validating configuration from file: %w", err)
	}
	return nil
}

// FileExists returns whether a wisteria configuration file exists at the default path.
func (c *Config) FileExists() (bool, error) {
	defaultPath, err := DefaultConfigFilePath()
	if err != nil {
		return false, fmt.Errorf("unable to get default path: %w", err)
	}
	if _, err := os.Stat(defaultPath); err != nil {
		return false, fmt.Errorf("unable to confirm existence of config file: %w", err)
	}
	return true, nil
}

// SaveTo persists this configuration within the given WriteCloser and then closes it.
func (c *Config) SaveTo(configFile io.Writer) error {
	encoder := json.NewEncoder(configFile)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed writing config file: %w", err)
	}
	return nil
}

func (c *Config) SaveToPath(configpath string) (err error) {
	if err := os.MkdirAll(filepath.Dir(configpath), 0755); err != nil {
		return fmt.Errorf("failed ensuring config directory exists: %w", err)
	}
	configFile, err := os.OpenFile(configpath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0664)
	if err != nil {
		return fmt.Errorf("couldn't save config file %s: %w", configpath, err)
	}
	defer func() {
		if closeErr := configFile.Close(); closeErr != nil {
			err = fmt.Errorf("failed closing config file: %w", err)
		}
	}()
	return c.SaveTo(configFile)
}

// Validate errors if the configuration is invalid
func (c *Config) Validate() error {
	switch {
	case c.PGPUser == "":
		return fmt.Errorf("PGPUser must be set")
	case c.IdentityID == "":
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
func (c *Config) Builder(store forest.Store) (*forest.Builder, error) {
	var (
		signer forest.Signer
		err    error
	)
	if c.PGPUser != "" {
		signer, err = forest.NewGPGSigner(c.PGPUser)
		asGPG := signer.(*forest.GPGSigner)
		asGPG.Rewriter = func(cmd *exec.Cmd) error {
			cmd.Stderr = log.Writer()
			return nil
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	identity, err := c.IdentityNode(store)
	if err != nil {
		return nil, fmt.Errorf("failed getting identity node: %w", err)
	}
	return forest.As(identity, signer), nil
}

func (c *Config) IdentityNode(store forest.Store) (*forest.Identity, error) {
	identityID := &fields.QualifiedHash{}
	if err := identityID.UnmarshalText([]byte(c.IdentityID)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IdentityID %s into QualifiedHash: %w", c.IdentityID, err)
	}
	identity, has, err := store.GetIdentity(identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity %s: %w", c.IdentityID, err)
	} else if !has {
		return nil, fmt.Errorf("store does not contain identity %s", c.IdentityID)
	}
	return identity.(*forest.Identity), nil
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
	In  *bufio.Reader
}

func NewStdoutPrompter(in io.Reader, out io.Writer) *StdoutPrompter {
	return &StdoutPrompter{
		Out: out,
		In:  bufio.NewReader(in),
	}
}

// Choose asks the user to choose from among a list of options. The formatter
// function is used to display each option to the user
func (s *StdoutPrompter) Choose(prompt string, slice []interface{}, formatter func(element interface{}) string) (choice interface{}, err error) {
	if len(slice) < 1 {
		return nil, fmt.Errorf("Cannot choose from empty option list")
	}
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
		str, err := s.In.ReadString("\n"[0])
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
// The identity will be stored in the provided forest.Store implementation
func (w *Wizard) ConfigureNewIdentity(store forest.Store) error {
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
	gpgPath, err := forest.FindGPG()
	if err != nil {
		w.Display(installGPGMessage)
		return fmt.Errorf("Failed finding gpg installation: %v", err)
	}
	if secKey.(string) == createNewOption {
		w.Display(fmt.Sprintf("\nTo create a new key, run:\n\n%s --generate-key\n\nRe-run %v when you've done that.\n", gpgPath, os.Args[0]))
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
	identity, err := forest.NewIdentity(signer, username, []byte{})
	if err != nil {
		return fmt.Errorf("Failed to create identity: %v", err)
	}
	if err := store.Add(identity); err != nil {
		return fmt.Errorf("Error saving new identity %s: %v", identity.ID(), err)
	}
	w.IdentityID = identity.ID().String()
	return nil
}

// ConfigureIdentity sets up an identity in the Wizard's config. It creates a new one
// if the user requests it.
func (w *Wizard) ConfigureIdentity(store forest.Store) error {
	count := 1024
	identities, err := store.Recent(fields.NodeTypeIdentity, count)
	if err != nil {
		return fmt.Errorf("failed looking up recent identities: %w", err)
	}
	// make sure we get *all* identities
	for len(identities) == count {
		count *= 2
		identities, err = store.Recent(fields.NodeTypeIdentity, count)
		if err != nil {
			return fmt.Errorf("failed looking up recent identities: %w", err)
		}
	}

	asGeneric := make([]interface{}, len(identities))
	for i := range identities {
		asGeneric[i] = identities[i]
	}
	// ensure that we have a typed nil to represent a the choice to create a new identity
	var makeNew *forest.Identity = nil
	asGeneric = append(asGeneric, makeNew)
	choiceInterface, err := w.Choose("Please choose an identity:", asGeneric, func(i interface{}) string {
		id := i.(*forest.Identity)
		if id == nil {
			return "create a new identity"
		}
		idString := id.ID().String()
		return fmt.Sprintf("%-16s %60s", string(id.Name.Blob), idString)
	})
	if err != nil {
		return fmt.Errorf("Error reading user response: %v", err)
	}

	choice := choiceInterface.(*forest.Identity)
	if choice != nil {
		w.IdentityID = choice.ID().String()
		return nil
	}

	return w.ConfigureNewIdentity(store)
}

const installGPGMessage = "This program requires GPG to run. Please install GPG and restart. https://gnupg.org/"

// Run populates the config by asking the user for information and
// inferring from the runtime environment
func (w *Wizard) Run(store forest.Store) error {
	_, err := forest.FindGPG()
	if err != nil {
		w.Display(installGPGMessage)
		return fmt.Errorf("Cannot configure without GPG: %v", err)
	}
	err = w.ConfigureIdentity(store)
	if err != nil {
		return fmt.Errorf("Error configuring user identity: %v", err)
	}
	identity, err := w.IdentityNode(store)
	if err != nil {
		return fmt.Errorf("Error getting identity node: %w", err)
	}
	key, err := identity.PublicKey.AsEntity()
	if err != nil {
		return fmt.Errorf("Error extracting key: %v", err)
	}
	pgpIds := []string{}
	for keyID := range key.Identities {
		pgpIds = append(pgpIds, keyID)
	}
	w.PGPUser = pgpIds[0]
	return nil
}
