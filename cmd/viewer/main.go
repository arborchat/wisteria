package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/rivo/tview"

	forest "git.sr.ht/~whereswaldon/forest-go"
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

func readAllInto(store forest.Store) ([]forest.Node, error) {
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
	var nodes []forest.Node
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
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func main() {
	store := forest.NewMemoryStore()
	nodes, err := readAllInto(store)
	if err != nil {
		log.Fatal(err)
	}
	if err := render(nodes, store); err != nil {
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

func render(nodes []forest.Node, store forest.Store) error {
	history := tview.NewTextView()
	for _, n := range nodes {
		if err := writeNode(history, n, store); err != nil {
			return err
		}
	}
	app := tview.NewApplication()
	return app.SetRoot(history, true).Run()
}
