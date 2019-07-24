package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

func index(element *fields.QualifiedHash, group []*fields.QualifiedHash) int {
	for i, current := range group {
		if element.Equals(current) {
			return i
		}
	}
	return -1
}

func in(element *fields.QualifiedHash, group []*fields.QualifiedHash) bool {
	return index(element, group) >= 0
}

func nth(input string, n int) rune {
	for i, r := range input {
		if i == n {
			return r
		}
	}
	return '?'
}

type NodeList []*forest.Reply

func (h NodeList) Sort() {
	sort.SliceStable(h, func(i, j int) bool {
		return h[i].Created < h[j].Created
	})
}

func (h NodeList) IndexForID(id *fields.QualifiedHash) int {
	for i, n := range h {
		if n.ID().Equals(id) {
			return i
		}
	}
	return -1
}

type Archive struct {
	NodeList
	forest.Store
}

func (a *Archive) Read(r io.ReadCloser) error {
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	node, err := forest.UnmarshalBinaryNode(b)
	if err != nil {
		return err
	}
	if err := a.Store.Add(node); err != nil {
		return err
	}
	if r, ok := node.(*forest.Reply); ok {
		a.NodeList = append(a.NodeList, r)
	}
	return nil

}

func readAllInto(store forest.Store) (*Archive, error) {
	workdir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dir, err := os.Open(workdir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	var nodes []*forest.Reply
	archive := &Archive{NodeList: nodes, Store: store}
	for _, name := range names {
		nodeFile, err := os.Open(name)
		if err != nil {
			log.Println(err)
			continue
		}
		err = archive.Read(nodeFile)
		if err != nil {
			log.Printf("Failed parsing %s: %v", nodeFile.Name(), err)
			continue
		}
	}
	return archive, nil
}

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

func (v *Archive) DescendantsOf(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	descendants := make([]*fields.QualifiedHash, 0)
	directChildren := []*fields.QualifiedHash{id}

	for len(directChildren) > 0 {
		target := directChildren[0]
		directChildren = directChildren[1:]
		for _, node := range v.NodeList {
			if node.ParentID().Equals(target) {
				descendants = append(descendants, node.ID())
				directChildren = append(directChildren, node.ID())
			}
		}
	}
	return descendants, nil
}

type HistoryView struct {
	*Archive
	Current    *fields.QualifiedHash
	rendered   []string
	lineStyles []tcell.Style
}

var _ views.CellModel = &HistoryView{}

func (v *HistoryView) EnsureCurrent() {
	if v.Current == nil {
		v.Current = v.NodeList[0].ID()
	}
}

func (v *HistoryView) CursorDown() {
	v.EnsureCurrent()
	currIndex := v.NodeList.IndexForID(v.Current)
	switch {
	case currIndex < 0:
		return
	case currIndex >= 0 && currIndex < len(v.NodeList)-1:
		v.Current = v.NodeList[currIndex+1].ID()
	}
	_ = v.Render()
}

func (v *HistoryView) CursorUp() {
	v.EnsureCurrent()
	currIndex := v.NodeList.IndexForID(v.Current)
	switch {
	case currIndex < 0:
		return
	case currIndex > 0:
		v.Current = v.NodeList[currIndex-1].ID()
	}
	_ = v.Render()
}

/*
func (v *HistoryView) Current() (*forest.Reply, *forest.Identity, error) {
	node, has, err := v.Get(v.Current)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, err
	} else if reply, ok := node.(*forest.Reply); !ok {
		return nil, nil, fmt.Errorf("Current node is not a reply: %v", node)
	}

}
*/

func (v *HistoryView) Render() error {
	v.rendered = []string{}
	v.lineStyles = []tcell.Style{}
	v.EnsureCurrent()
	ancestry, err := v.AncestryOf(v.Current)
	if err != nil {
		return err
	}
	descendants, err := v.DescendantsOf(v.Current)
	if err != nil {
		return err
	}
	for _, n := range v.NodeList {
		config := renderConfig{}
		if n.ID().Equals(v.Current) {
			config.state = current
		} else if in(n.ID(), ancestry) {
			config.state = ancestor
		} else if in(n.ID(), descendants) {
			config.state = descendant
		}
		asString, style, err := renderNode(n, v.Store, config)
		if err != nil {
			return err
		}
		v.rendered = append(v.rendered, asString...)
		for i := 0; i < len(asString); i++ {
			v.lineStyles = append(v.lineStyles, style)
		}
	}
	return nil
}

func (v *HistoryView) GetCell(x, y int) (rune, tcell.Style, []rune, int) {
	if y < len(v.rendered) && x < len(v.rendered[y]) {
		return nth(v.rendered[y], x), v.lineStyles[y], nil, 1
	}
	return ' ', tcell.StyleDefault, nil, 1
}

func (v *HistoryView) GetBounds() (int, int) {
	width := 0
	for _, line := range v.rendered {
		if len(line) > width {
			width = len(line)
		}
	}
	height := len(v.rendered)
	return height - 1, width - 1
}

func (v *HistoryView) SetCursor(x, y int) {
}

func (v *HistoryView) GetCursor() (int, int, bool, bool) {
	return 0, 0, false, false
}

func (v *HistoryView) MoveCursor(offx, offy int) {
}

type HistoryWidget struct {
	*HistoryView
	*views.CellView
	*views.Application
}

var _ views.Widget = &HistoryWidget{}

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
			file, err := ioutil.TempFile("", "arbor-msg")
			if err != nil {
				log.Println(err)
			}
			_, err = file.Write([]byte(fmt.Sprintf("# replying to %s\n", "")))
			if err != nil {
				file.Close()
				log.Println(err)
			}
			file.Close()
			editor := exec.Command("gnome-terminal", "-q", "--", os.ExpandEnv("$EDITOR"), file.Name())
			editor.Stdin = os.Stdin
			editor.Stdout = os.Stdout
			editor.Stderr = os.Stderr
			if err := editor.Run(); err != nil {
				log.Println(err)
			}

		case tcell.KeyRune:
			// break if it's a normal keypress
		default:
			return false
		}
		switch keyEvent.Rune() {
		case 'j':
			v.CursorDown()
			return true
		case 'k':
			v.CursorUp()
			return true
		}
	}
	return false
}

func main() {
	store := forest.NewMemoryStore()
	history, err := readAllInto(store)
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
	cv := views.NewCellView()
	cv.SetModel(historyView)
	app := new(views.Application)
	hw := &HistoryWidget{
		historyView,
		cv,
		app,
	}
	app.SetRootWidget(hw)

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if err := Watch(cwd, func(filename string) {
		app.PostFunc(func() {
			file, err := os.Open(filename)
			if err != nil {
				log.Println(err)
			}
			defer file.Close()
			err = historyView.Read(file)
			if err != nil {
				log.Println(err)
			}
			historyView.Sort()
			err = historyView.Render()
			if err != nil {
				log.Println(err)
			}
			app.Update()
		})
	}); err != nil {
		log.Fatal(err)
	}

	if e := app.Run(); e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		os.Exit(1)
	}
}

func Watch(dir string, handler func(filename string)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					handler(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	err = watcher.Add(dir)
	if err != nil {
		return err
	}
	return nil
}

type nodeState uint

const (
	none nodeState = iota
	ancestor
	descendant
	current
)

type renderConfig struct {
	state nodeState
}

func renderNode(node forest.Node, store forest.Store, config renderConfig) ([]string, tcell.Style, error) {
	var (
		ancestorColor   = tcell.StyleDefault.Foreground(tcell.ColorYellow)
		descendantColor = tcell.StyleDefault.Foreground(tcell.ColorGreen)
		currentColor    = tcell.StyleDefault.Foreground(tcell.ColorRed)
	)
	var out []string
	var style tcell.Style
	switch n := node.(type) {
	case *forest.Reply:
		author, present, err := store.Get(&n.Author)
		if err != nil {
			return nil, tcell.StyleDefault, err
		} else if !present {
			return nil, tcell.StyleDefault, fmt.Errorf("Node %v is not in the store", n.Author)
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
		out = append(out, strings.Split(rendered, "\n")...)
	}
	return out, style, nil
}
