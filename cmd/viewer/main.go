package main

import (
	"bufio"
	"bytes"
	"encoding"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"github.com/pkg/profile"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// save encodes the binary form of the given BinaryMarshaler into `w`
func save(w io.Writer, node encoding.BinaryMarshaler) error {
	b, err := node.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// saveAs stores the binary form of the given BinaryMarshaler into a new file called `name`
func saveAs(name string, node encoding.BinaryMarshaler) error {
	outfile, err := os.Create(name)
	if err != nil {
		return err
	}
	log.Printf("saving to %s", outfile.Name())
	defer outfile.Close()

	return save(outfile, node)
}

// index returns the index of `element` within `group`, or -1 if it is not present
func index(element *fields.QualifiedHash, group []*fields.QualifiedHash) int {
	for i, current := range group {
		if element.Equals(current) {
			return i
		}
	}
	return -1
}

// in returns whether `element` is in `group`
func in(element *fields.QualifiedHash, group []*fields.QualifiedHash) bool {
	return index(element, group) >= 0
}

// nth returns the `n`th rune in the input string. Note that this is not the same as the
// Nth byte of data, as unicode runes can take multiple bytes.
func nth(input string, n int) rune {
	for i, r := range input {
		if i == n {
			return r
		}
	}
	return '?'
}

// ReplyList holds a sortable list of replies
type ReplyList []*forest.Reply

func (h ReplyList) Sort() {
	sort.SliceStable(h, func(i, j int) bool {
		return h[i].Created < h[j].Created
	})
}

// IndexForID returns the position of the node with the given `id` inside of the ReplyList,
// or -1 if it is not present.
func (h ReplyList) IndexForID(id *fields.QualifiedHash) int {
	for i, n := range h {
		if n.ID().Equals(id) {
			return i
		}
	}
	return -1
}

// Archive manages a group of known arbor nodes
type Archive struct {
	ReplyList
	forest.Store
}

// NodesFromDir reads all files in the directory and returns all of them that contained
// arbor nodes in a slice
func NodesFromDir(dirname string) []forest.Node {
	dir, err := os.Open(dirname)
	if err != nil {
		return nil
	}
	defer dir.Close()
	names, err := dir.Readdirnames(0)
	if err != nil {
		return nil
	}
	var nodes []forest.Node
	for _, name := range names {
		b, err := ioutil.ReadFile(name)
		if err != nil {
			log.Println(err)
			continue
		}
		node, err := forest.UnmarshalBinaryNode(b)
		if err != nil {
			log.Printf("Failed parsing %s: %v", name, err)
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// NewArchiveFromDir creates an archive populated with the contents of `dirname` and
// using `store` as the storage back-end.
func NewArchiveFromDir(dirname string, store forest.Store) (*Archive, error) {
	nodes := NodesFromDir(dirname)
	var replies []*forest.Reply
	archive := &Archive{ReplyList: replies, Store: store}
	for _, n := range nodes {
		err := archive.Add(n)
		if err != nil {
			return nil, err
		}
	}
	return archive, nil
}

// Add accepts an arbor node and stores it in the Archive. If it is
// a Reply node, it will be added to the ReplyList
func (a *Archive) Add(node forest.Node) error {
	if err := a.Store.Add(node); err != nil {
		return err
	}
	if r, ok := node.(*forest.Reply); ok {
		a.ReplyList = append(a.ReplyList, r)
	}
	return nil

}

// AncestryOf returns the IDs of all known ancestors of the node with the given `id`
func (v *Archive) AncestryOf(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	node, present, err := v.Store.Get(id)
	if err != nil {
		return nil, err
	} else if !present {
		return []*fields.QualifiedHash{}, nil
	}
	ancestors := make([]*fields.QualifiedHash, 0, node.TreeDepth())
	next := node.ParentID()
	for !next.Equals(fields.NullHash()) {
		parent, present, err := v.Store.Get(next)
		if err != nil {
			return nil, err
		} else if !present {
			return ancestors, nil
		}
		ancestors = append(ancestors, next)
		next = parent.ParentID()
	}
	return ancestors, nil
}

// DescendantsOf returns the IDs of all known descendants of the node with the given `id`
func (v *Archive) DescendantsOf(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	descendants := make([]*fields.QualifiedHash, 0)
	directChildren := []*fields.QualifiedHash{id}

	for len(directChildren) > 0 {
		target := directChildren[0]
		directChildren = directChildren[1:]
		for _, node := range v.ReplyList {
			if node.ParentID().Equals(target) {
				descendants = append(descendants, node.ID())
				directChildren = append(directChildren, node.ID())
			}
		}
	}
	return descendants, nil
}

// RenderedLine represents a single line of text in the terminal UI
type RenderedLine struct {
	ID    *fields.QualifiedHash
	Style tcell.Style
	Text  string
}

// HistoryView models the visible contents of the chat history. It implements tcell.CellModel
type HistoryView struct {
	*Archive
	rendered []RenderedLine
	Cursor   struct {
		X, Y int
	}
}

var _ views.CellModel = &HistoryView{}

// CurrentID returns the ID of the currently-selected node
func (v *HistoryView) CurrentID() *fields.QualifiedHash {
	if len(v.rendered) > v.Cursor.Y {
		return v.rendered[v.Cursor.Y].ID
	} else if len(v.Archive.ReplyList) > 0 {
		return v.Archive.ReplyList[0].ID()
	}
	return fields.NullHash()
}

// CurrentReply returns the currently-selected node
func (v *HistoryView) CurrentReply() (*forest.Reply, error) {
	node, has, err := v.Get(v.CurrentID())
	if err != nil {
		return nil, err
	} else if !has {
		return nil, err
	} else if reply, ok := node.(*forest.Reply); !ok {
		return nil, fmt.Errorf("Current node is not a reply: %v", node)
	} else {
		return reply, nil
	}

}

// Render recomputes the contents of this view, taking any changes in the nodes in the underlying
// Archive and position of the cursor into account.
func (v *HistoryView) Render() error {
	currentID := v.CurrentID()
	currentIDText, _ := currentID.MarshalString()
	log.Printf("Starting Render() with %s as current", currentIDText)
	v.rendered = []RenderedLine{}
	ancestry, err := v.AncestryOf(currentID)
	if err != nil {
		return err
	}
	log.Printf("len(ancestry) = %d", len(ancestry))
	descendants, err := v.DescendantsOf(currentID)
	if err != nil {
		return err
	}
	log.Printf("len(descendants) = %d", len(descendants))
	for _, n := range v.ReplyList {
		config := renderConfig{}
		if n.ID().Equals(currentID) {
			config.state = current
		} else if in(n.ID(), ancestry) {
			config.state = ancestor
		} else if in(n.ID(), descendants) {
			config.state = descendant
		}
		lines, err := renderNode(n, v.Store, config)
		if err != nil {
			return err
		}
		v.rendered = append(v.rendered, lines...)
	}
	return nil
}

// GetCell returns the contents of a single cell of the view
func (v *HistoryView) GetCell(x, y int) (cell rune, style tcell.Style, combining []rune, width int) {
	cell, style, combining, width = ' ', tcell.StyleDefault, nil, 1
	if y < len(v.rendered) && x < len(v.rendered[y].Text) {
		cell, style, combining, width = nth(v.rendered[y].Text, x), v.rendered[y].Style, nil, 1
	}
	if v.Cursor.X == x && v.Cursor.Y == y {
		style = tcell.StyleDefault.Reverse(true)
	}
	return
}

// GetBounds returns the dimensions of the view
func (v *HistoryView) GetBounds() (int, int) {
	width := 0
	for _, line := range v.rendered {
		if len(line.Text) > width {
			width = len(line.Text)
		}
	}
	height := len(v.rendered) + MaxEmtpyVisibleLines
	return width, height
}

// SetCursor warps the cursor to the given coordinates
func (v *HistoryView) SetCursor(x, y int) {
	v.Cursor.X = x
	v.Cursor.Y = y
	if err := v.Render(); err != nil {
		log.Println("Error rendering after SetCursor():", err)
	}
}

// GetCursor returns the position of the cursor, whether it is enabled, and whether it is hidden
func (v *HistoryView) GetCursor() (int, int, bool, bool) {
	return v.Cursor.X, v.Cursor.Y, true, false
}

const MaxEmtpyVisibleLines = 15

// MoveCursor moves the cursor relative to its current position
func (v *HistoryView) MoveCursor(offx, offy int) {
	w, h := v.GetBounds()
	if v.Cursor.X+offx >= 0 {
		if v.Cursor.X+offx < w {
			v.Cursor.X += offx
		} else {
			v.Cursor.X = w - 1
		}
	}
	if v.Cursor.Y+offy >= 0 {
		if v.Cursor.Y+offy < h {
			v.Cursor.Y += offy
		} else {
			v.Cursor.Y = h - 1
		}
	}
	if err := v.Render(); err != nil {
		log.Printf("Error during post-cursor move render: %v", err)
	}
}

// HistoryWidget is the controller for the chat history TUI
type HistoryWidget struct {
	*HistoryView
	*CellView
	*views.Application
	*forest.Builder
	*Config
}

var _ views.Widget = &HistoryWidget{}

func (v *HistoryWidget) ReadMessageFile(filename string) {
	v.Application.PostFunc(func() {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Println(err)
			return
		}
		node, err := forest.UnmarshalBinaryNode(b)
		if err != nil {
			log.Println(err)
			return
		}
		err = v.Add(node)
		if err != nil {
			log.Println(err)
			return
		}
		v.Sort()
		err = v.Render()
		if err != nil {
			log.Println(err)
			return
		}
		v.Application.Update()
	})
}

func (v *HistoryWidget) HandleEvent(event tcell.Event) bool {
	if v.CellView.HandleEvent(event) {
		return true
	}
	switch keyEvent := event.(type) {
	case *tcell.EventKey:
		switch keyEvent.Key() {
		case tcell.KeyCtrlC:
			v.Application.Quit()
		case tcell.KeyEnter:
			reply, err := v.CurrentReply()
			if err != nil {
				log.Println(err)
				return false
			}
			msg := strings.Join(strings.Split(string(reply.Content.Blob), "\n"), "\n#")
			file, err := ioutil.TempFile("", "arbor-msg")
			if err != nil {
				log.Println(err)
				return false
			}
			_, err = file.Write([]byte(fmt.Sprintf("# replying to %s\n", msg)))
			if err != nil {
				file.Close()
				log.Println(err)
				return false
			}
			file.Close()
			log.Print("starting editor")
			if err := v.Config.EditFile(file.Name()).Run(); err != nil {
				log.Println(err)
				return false
			}
			log.Print("editor done")
			replyContent, err := ioutil.ReadFile(file.Name())
			if err != nil {
				log.Println(err)
				return false
			}
			log.Print(string(replyContent))
			reply, err = v.NewReply(reply, stripCommentLines(string(replyContent)), "")
			if err != nil {
				log.Println(err)
				return false
			}
			outfile, err := reply.ID().MarshalString()
			if err != nil {
				log.Println(err)
				return false
			}
			err = saveAs(outfile, reply)
			if err != nil {
				log.Println(err)
				return false
			}

		case tcell.KeyRune:
			// break if it's a normal keypress
		default:
			return false
		}
		switch keyEvent.Rune() {
		case 'h':
			v.MoveCursor(-1, 0)
			v.MakeCursorVisible()
			return true
		case 'j':
			v.MoveCursor(0, 1)
			v.MakeCursorVisible()
			return true
		case 'k':
			v.MoveCursor(0, -1)
			v.MakeCursorVisible()
			return true
		case 'l':
			v.MoveCursor(1, 0)
			v.MakeCursorVisible()
			return true
		}
	}
	return false
}

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

func main() {
	config := NewConfig()
	defer profile.Start(profile.ProfilePath(config.RuntimeDirectory)).Stop()
	logPath := path.Join(config.RuntimeDirectory, "viewer.log")
	log.Println("Logging to", logPath)
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	flag.StringVar(&config.PGPUser, "gpguser", "", "gpg user to sign new messages with")
	flag.StringVar(&config.PGPKey, "key", "", "PGP key to sign messages with")
	flag.StringVar(&config.IdentityName, "identity", "", "arbor identity node to sign with")
	flag.Parse()
	if flag.NArg() > 0 {
		config.EditorCmd = flag.Args()
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if err := RunWizard(cwd, config); err != nil {
		log.Fatal("Error running configuration wizard", err)
	}
	if err := config.Validate(); err != nil {
		log.Fatal("Error validating configuration:", err)
	}
	builder, err := config.Builder()
	if err != nil {
		log.Fatal("Unable to construct builder using configuration:", err)
	}
	store := forest.NewMemoryStore()
	history, err := NewArchiveFromDir(cwd, store)
	if err != nil {
		log.Fatal(err)
	}
	history.Sort()
	historyView := &HistoryView{
		Archive: history,
	}
	if err := historyView.Render(); err != nil {
		log.Fatal(err)
	}
	cv := NewCellView()
	cv.SetModel(historyView)
	cv.MakeCursorVisible()
	app := new(views.Application)
	hw := &HistoryWidget{
		historyView,
		cv,
		app,
		builder,
		config,
	}
	app.SetRootWidget(hw)

	if _, err := Watch(cwd, hw.ReadMessageFile); err != nil {
		log.Fatal(err)
	} else {
		//		defer watcher.Close()
	}

	if e := app.Run(); e != nil {
		log.Println(e.Error())
		os.Exit(1)
	}
}

// Watch watches for file creation events in `dir`. It executes `handler` on each event.
func Watch(dir string, handler func(filename string)) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create != 0 {
					log.Println("Got create event for", event.Name)
					handler(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("Got watch error", err)
					return
				}
			}
		}
	}()
	err = watcher.Add(dir)
	if err != nil {
		return nil, err
	}
	log.Println("Watching", dir)
	return watcher, nil
}

// nodeState represents possible render states for nodes
type nodeState uint

const (
	none nodeState = iota
	ancestor
	descendant
	current
)

// renderConfig holds information about how a particular node should be rendered
type renderConfig struct {
	state nodeState
}

// renderNode transforms `node` into a slice of rendered lines, using `store` to look up nodes referenced
// by `node` and `config` to make style choices.
func renderNode(node forest.Node, store forest.Store, config renderConfig) ([]RenderedLine, error) {
	var (
		ancestorColor   = tcell.StyleDefault.Foreground(tcell.ColorYellow)
		descendantColor = tcell.StyleDefault.Foreground(tcell.ColorGreen)
		currentColor    = tcell.StyleDefault.Foreground(tcell.ColorRed)
	)
	idstring, _ := node.ID().MarshalString()
	log.Printf("%s => %d", idstring, config.state)
	var out []RenderedLine
	var style tcell.Style
	switch n := node.(type) {
	case *forest.Reply:
		author, present, err := store.Get(&n.Author)
		if err != nil {
			return nil, err
		} else if !present {
			return nil, fmt.Errorf("Node %v is not in the store", n.Author)
		}
		asIdent := author.(*forest.Identity)
		switch config.state {
		case ancestor:
			style = ancestorColor
		case descendant:
			style = descendantColor
		case current:
			style = currentColor
		default:
			style = tcell.StyleDefault
		}
		rendered := fmt.Sprintf("%s: %s", string(asIdent.Name.Blob), string(n.Content.Blob))
		// drop all trailing newline characters
		for rendered[len(rendered)-1] == "\n"[0] {
			rendered = rendered[:len(rendered)-1]
		}
		for _, line := range strings.Split(rendered, "\n") {
			out = append(out, RenderedLine{
				ID:    n.ID(),
				Style: style,
				Text:  line,
			})
		}
	}
	return out, nil
}

// stripCommentLines removes all lines in `input` that begin with "#"
func stripCommentLines(input string) string {
	lines := strings.Split(input, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
