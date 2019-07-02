package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/rivo/tview"

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
			log.Println(err)
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

type HistoryView struct {
	History
	forest.Store
	Current fields.QualifiedHash
	*tview.TextView
}

func (v *HistoryView) Render() error {
	for _, n := range v.History {
		if err := writeNode(v, n, v); err != nil {
			return err
		}
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
		Store:    store,
		TextView: tview.NewTextView(),
		History:  nodes,
	}
	app := tview.NewApplication()

	if err := app.SetRoot(history, true).Run(); err != nil {
		log.Fatal(err)
	}
}

func writeNode(w io.Writer, node forest.Node, store forest.Store) error {
	var out string
	switch n := node.(type) {
	case *forest.Reply:
		author, present, err := store.Get(&n.Author)
		if err != nil {
			return err
		} else if !present {
			return fmt.Errorf("Node %v is not in the store", n.Author)
		}
		asIdent := author.(*forest.Identity)
		out = fmt.Sprintf("%s: %s\n", string(asIdent.Name.Blob), string(n.Content.Blob))
	}
	_, err := w.Write([]byte(out))
	return err
}
