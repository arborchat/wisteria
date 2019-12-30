package main

import (
	"fmt"
	"log"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// RenderedLine represents a single line of text in the terminal UI
type RenderedLine struct {
	ID    *fields.QualifiedHash
	Style tcell.Style
	Text  string
}

// HistoryView models the visible contents of the chat history. It implements tcell.CellModel
type HistoryView struct {
	*Archive
	FilterID, SelectedReplyID *fields.QualifiedHash
	rendered                  []RenderedLine
	Cursor                    struct {
		X, Y int
	}
}

var _ views.CellModel = &HistoryView{}

// CurrentID returns the ID of the currently-selected node
func (v *HistoryView) CurrentID() *fields.QualifiedHash {
	if v.SelectedReplyID != nil {
		return v.SelectedReplyID
	}
	return fields.NullHash()
}

// UpdateCurrentID recomputes the currently-selected message id based on the
// current position of the cursor.
func (v *HistoryView) UpdateCurrentID() {
	if len(v.rendered) > v.Cursor.Y && v.Cursor.Y > -1 {
		v.SelectedReplyID = v.rendered[v.Cursor.Y].ID
	} else if len(v.Archive.ReplyList) > 0 {
		v.SelectedReplyID = v.Archive.ReplyList[0].ID()
	} else {
		v.SelectedReplyID = nil
	}
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
	v.rendered = []RenderedLine{}
	ancestry, err := v.AncestryOf(currentID)
	if err != nil {
		return fmt.Errorf("failed looking up ancestry of %s: %w", currentID.String(), err)
	}
	descendants, err := v.DescendantsOf(currentID)
	if err != nil {
		return fmt.Errorf("failed looking up descendants of %s: %w", currentID.String(), err)
	}
	excludeMap := make(map[string]struct{})
	if v.FilterID != nil {
		filterAncestry, err := v.AncestryOf(v.FilterID)
		if err != nil {
			return fmt.Errorf("failed lookup up ancestry of filter node %s: %w", v.FilterID, err)
		}
		filterDescendants, err := v.DescendantsOf(v.FilterID)
		if err != nil {
			return fmt.Errorf("failed lookup up descendants of filter node %s: %w", v.FilterID, err)
		}
		excludeMap[v.FilterID.String()] = struct{}{}
		for _, id := range append(filterAncestry, filterDescendants...) {
			excludeMap[id.String()] = struct{}{}
		}
	}
	for _, n := range v.ReplyList {
		if v.FilterID != nil {
			if _, matchesFilter := excludeMap[n.ID().String()]; !matchesFilter {
				// skip nodes that don't match current filter
				continue
			}
		}
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
			log.Printf("failed rendering %s: %v", n.ID().String(), err)
			continue
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
	height := len(v.rendered) + MaxEmptyVisibleLines
	return width, height
}

// SetCursor warps the cursor to the given coordinates
func (v *HistoryView) SetCursor(x, y int) {
	v.Cursor.X = x
	v.Cursor.Y = y
	v.UpdateCurrentID()
	if err := v.Render(); err != nil {
		log.Println("Error rendering after SetCursor():", err)
	}
}

// GetCursor returns the position of the cursor, whether it is enabled, and whether it is hidden
func (v *HistoryView) GetCursor() (int, int, bool, bool) {
	return v.Cursor.X, v.Cursor.Y, true, false
}

const MaxEmptyVisibleLines = 15

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
	v.UpdateCurrentID()
	if err := v.Render(); err != nil {
		log.Printf("Error during post-cursor move render: %v", err)
	}
}

// SelectLastLine warps the cursor to the final line of rendered text
func (v *HistoryView) SelectLastLine() {
	_, h := v.GetBounds()
	v.SetCursor(0, h-1-MaxEmptyVisibleLines)
}

// FilterOnCurrent sets the ID of the node to filter on to be the ID of the
// currently-selected node.
func (v *HistoryView) FilterOnCurrent() {
	v.FilterID = v.CurrentID()
	v.moveCursorToSelected()
}

func (v *HistoryView) moveCursorToSelected() {
	if err := v.Render(); err != nil {
		log.Printf("Error during first post-clear render: %v", err)
	}
	y := 0
	for i, renderedLine := range v.rendered {
		if renderedLine.ID.Equals(v.SelectedReplyID) {
			y = i
		}
	}
	v.SetCursor(v.Cursor.X, y)
}

// ClearFilter erases the filter on the view to show all nodes again.
func (v *HistoryView) ClearFilter() {
	v.FilterID = nil
	v.moveCursorToSelected()
}

// ToggleFilter clears any filter that is set, but sets the current message
// to be the filter if there is no current filter set.
func (v *HistoryView) ToggleFilter() {
	if v.FilterID != nil {
		v.ClearFilter()
		return
	}
	v.FilterOnCurrent()
}
