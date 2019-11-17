package main

import (
	"fmt"
	"strings"
	"time"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"github.com/gdamore/tcell"
)

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
