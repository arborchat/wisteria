package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

func readInto(r io.ReadCloser, store forest.Store) (forest.Node, error) {
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	node, err := forest.UnmarshalBinaryNode(b)
	if err != nil {
		return nil, err
	}
	if err := store.Add(node); err != nil {
		return nil, err
	}
	return node, nil

}

type History []*forest.Reply

func readAllInto(store forest.Store) (History, error) {
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
	for _, name := range names {
		nodeFile, err := os.Open(name)
		if err != nil {
			log.Println(err)
			continue
		}
		node, err := readInto(nodeFile, store)
		if err != nil {
			log.Printf("Failed parsing %s: %v", nodeFile.Name(), err)
			continue
		}
		if r, ok := node.(*forest.Reply); ok {
			nodes = append(nodes, r)
		}
	}
	return nodes, nil
}

func (h History) Sort() {
	sort.SliceStable(h, func(i, j int) bool {
		return h[i].Created < h[j].Created
	})
}

func (h History) IndexForID(id *fields.QualifiedHash) int {
	for i, n := range h {
		if n.ID().Equals(id) {
			return i
		}
	}
	return -1
}

type HistoryView struct {
	History
	forest.Store
	Current    *fields.QualifiedHash
	rendered   []string
	lineStyles []tcell.Style
}

func (v *HistoryView) AncestryOf(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
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

func (v *HistoryView) DescendantsOf(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	descendants := make([]*fields.QualifiedHash, 0)
	directChildren := []*fields.QualifiedHash{id}

	for len(directChildren) > 0 {
		target := directChildren[0]
		directChildren = directChildren[1:]
		for _, node := range v.History {
			if node.ParentID().Equals(target) {
				descendants = append(descendants, node.ID())
				directChildren = append(directChildren, node.ID())
			}
		}
	}
	return descendants, nil
}

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

func (v *HistoryView) EnsureCurrent() {
	if v.Current == nil {
		v.Current = v.History[0].ID()
	}
}

func (v *HistoryView) CursorDown() {
	v.EnsureCurrent()
	currIndex := v.History.IndexForID(v.Current)
	switch {
	case currIndex < 0:
		return
	case currIndex >= 0 && currIndex < len(v.History)-1:
		v.Current = v.History[currIndex+1].ID()
	}
	_ = v.Render()
}

func (v *HistoryView) CursorUp() {
	v.EnsureCurrent()
	currIndex := v.History.IndexForID(v.Current)
	switch {
	case currIndex < 0:
		return
	case currIndex > 0:
		v.Current = v.History[currIndex-1].ID()
	}
	_ = v.Render()
}

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
	for _, n := range v.History {
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

func nth(input string, n int) rune {
	for i, r := range input {
		if i == n {
			return r
		}
	}
	return '?'
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

func (v *HistoryView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		// break if it's a normal keypress
	default:
		return event
	}
	switch event.Rune() {
	case 'j':
		v.CursorDown()
	case 'k':
		v.CursorUp()
	default:
		return event
	}
	return nil
}

func main() {
	store := forest.NewMemoryStore()
	nodes, err := readAllInto(store)
	if err != nil {
		log.Fatal(err)
	}
	nodes.Sort()
	history := &HistoryView{
		Store:   store,
		History: nodes,
	}
	if err := history.Render(); err != nil {
		log.Fatal(err)
	}
	cv := views.NewCellView()
	cv.SetModel(history)
	app := new(views.Application)
	app.SetRootWidget(cv)
	if e := app.Run(); e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		os.Exit(1)
	}
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
