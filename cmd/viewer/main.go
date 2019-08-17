package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/0xAX/notificator"
	"github.com/fsnotify/fsnotify"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"github.com/pkg/profile"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

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
	if _, has, _ := a.Store.Get(node.ID()); has {
		return fmt.Errorf("Archive already contains %v", node)
	}
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

// SelectLastLine warps the cursor to the final line of rendered text
func (v *HistoryView) SelectLastLine() {
	_, h := v.GetBounds()
	v.SetCursor(0, h-1-MaxEmtpyVisibleLines)
}

// HistoryWidget is the controller for the chat history TUI
type HistoryWidget struct {
	*HistoryView
	*CellView
	*views.Application
	*forest.Builder
	*Config
	*notificator.Notificator
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
		if reply, ok := node.(*forest.Reply); ok {
			v.TryNotify(reply)
		}
	})
}

// TryNotify checks whether a desktop notification should be sent
// and attempts to send it
func (v *HistoryWidget) TryNotify(reply *forest.Reply) {
	username := strings.ToLower(string(v.Config.Identity.Name.Blob))
	messageText := strings.ToLower(string(reply.Content.Blob))
	if !strings.Contains(messageText, username) {
		return
	}
	author, has, err := v.Get(&reply.Author)
	if err != nil {
		log.Printf("Couldn't render desktop notification: %v", err)
		return
	} else if !has {
		log.Println("Couldn't render desktop notification: author information missing")
		return
	}
	log.Printf("Pushing notification: %v", v.Push("Arbor Mention from "+string(author.(*forest.Identity).Name.Blob), string(reply.Content.Blob), "", notificator.UR_NORMAL))
}

func (v *HistoryWidget) StartReply() error {
	reply, err := v.CurrentReply()
	if err != nil {
		return fmt.Errorf("couldn't determine current reply: %v", err)
	}
	msg := strings.Join(strings.Split(string(reply.Content.Blob), "\n"), "\n#")
	file, err := ioutil.TempFile("", "arbor-msg")
	if err != nil {
		return fmt.Errorf("couldn't create temporary file for reply: %v", err)
	}
	// ensure this file descriptor is closed
	file.Close()
	// populate the file, but keep it closed
	err = ioutil.WriteFile(file.Name(), []byte(fmt.Sprintf("# replying to %s\n", msg)), 0660)
	if err != nil {
		return fmt.Errorf("couldn't write template into temporary file: %v", err)
	}
	editor := v.Config.EditFile(file.Name())
	if err := editor.Start(); err != nil {
		return fmt.Errorf("failed to start editor command: %v", err)
	}
	go v.FinishReply(reply, file.Name(), editor)
	return nil
}

func (v *HistoryWidget) FinishReply(parent *forest.Reply, replyFileName string, editor *exec.Cmd) {
	if err := editor.Wait(); err != nil {
		log.Printf("Error waiting on editor command to finish: %v", err)
		log.Printf("There may be a partial message in %s", replyFileName)
		return
	}
	replyContent, err := ioutil.ReadFile(replyFileName)
	if err != nil {
		log.Printf("Error reading reply from %s: %v", replyFileName, err)
		return
	}
	replyContentString := strings.Trim(stripCommentLines(string(replyContent)), "\n")
	if len(replyContentString) == 0 {
		log.Println("Message is empty, not sending")
		return
	}
	reply, err := v.NewReply(parent, replyContentString, "")
	if err != nil {
		log.Printf("Error creating reply: %v", err)
		return
	}
	outfile, err := reply.ID().MarshalString()
	if err != nil {
		log.Printf("Error finding ID for reply: %v", err)
		return
	}
	err = saveAs(outfile, reply)
	if err != nil {
		log.Printf("Error saving to %s: %v", outfile, err)
		return
	}
	if err := os.Remove(replyFileName); err != nil {
		log.Printf("Error removing %s: %v", replyFileName, err)
		return
	}
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
			if err := v.StartReply(); err != nil {
				log.Printf("Error starting reply: %v", err)
				return true
			}
		case tcell.KeyRune:
			// break if it's a normal keypress
		default:
			return false
		}
		switch keyEvent.Rune() {
		case 'g':
			v.HistoryView.SetCursor(0, 0)
			v.MakeCursorVisible()
			return true
		case 'G':
			v.SelectLastLine()
			v.MakeCursorVisible()
			return true
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

func CheckNotify() {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("notify-send"); err != nil {
			log.Println("WARNING: desktop notifications require `notify-send` to be installed")
		}
	}
}

func main() {
	CheckNotify()
	var identityFile string
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
	flag.StringVar(&identityFile, "identity", "", "arbor identity node to sign with")
	flag.Parse()
	b, err := ioutil.ReadFile(identityFile)
	if err != nil {
	}
	config.Identity, err = forest.UnmarshalIdentity(b)
	if err != nil {
	}
	if flag.NArg() > 0 {
		config.EditorCmd = flag.Args()
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if config.Validate() != nil {
		wizard := &Wizard{
			Config:   config,
			Prompter: &StdoutPrompter{In: os.Stdin, Out: os.Stdout},
		}
		if err := wizard.Run(cwd); err != nil {
			log.Fatal("Error running configuration wizard:", err)
		}
		if err := config.Validate(); err != nil {
			log.Fatal("Error validating configuration:", err)
		}
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
	historyView.SelectLastLine() // start at bottom of history

	// set up desktop notifications
	notify := notificator.New(notificator.Options{
		AppName: "Arbor",
	})

	app := new(views.Application)
	hw := &HistoryWidget{
		historyView,
		cv,
		app,
		builder,
		config,
		notify,
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
				switch {
				case event.Op&fsnotify.Create != 0:
					log.Println("Got create event for", event.Name)
					handler(event.Name)
				case event.Op&fsnotify.Write != 0:
					log.Println("Got write event for", event.Name)
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
		ancestorColor         = tcell.StyleDefault.Foreground(tcell.ColorYellow)
		descendantColor       = tcell.StyleDefault.Foreground(tcell.ColorGreen)
		currentColor          = tcell.StyleDefault.Foreground(tcell.ColorRed)
		conversationRootColor = tcell.StyleDefault.Foreground(tcell.ColorTeal)
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
		timestamp := n.Created.Time().UTC()
		rendered := fmt.Sprintf("%s - %s:\n%s", timestamp.Format(time.Stamp), string(asIdent.Name.Blob), string(n.Content.Blob))
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
		if n.Depth == 1 {
			out[0].Style = conversationRootColor
		} else {
			out[0].Style = tcell.StyleDefault
		}
	}
	return out, nil
}
