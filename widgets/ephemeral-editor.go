package widgets

import (
	"log"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// EphemeralEditor is a layout that can summon and dismiss an Editor
// widget as needed. When it receives EventReplyRequest, it will create
// an Editor widget that will accept all key events before its PrimaryContent.
// Once that Editor emits EventSendRequest, it will dismiss the editor.
type EphemeralEditor struct {
	PrimaryContent    views.Widget
	Separator, Editor views.Widget
	EditorVisible     bool
	*views.BoxLayout
	views.WidgetWatchers
}

// NewEphemeralEditor creates a new layout with the given view as the
// primary content.
func NewEphemeralEditor(primary views.Widget) *EphemeralEditor {
	separator := views.NewTextBar()
	style := tcell.StyleDefault.Reverse(true)
	separator.SetStyle(style)
	separator.SetLeft("Type your reply below", style)
	e := &EphemeralEditor{
		PrimaryContent: primary,
		Editor:         NewEditor(),
		Separator:      separator,
		BoxLayout:      views.NewBoxLayout(views.Vertical),
	}
	e.BoxLayout.Watch(e)
	e.PrimaryContent.Watch(e)
	e.Editor.Watch(e)

	e.BoxLayout.AddWidget(e.PrimaryContent, 1.0)
	return e
}

// ShowEditor makes the editor visible.
func (e *EphemeralEditor) ShowEditor() {
	if !e.EditorVisible {
		e.BoxLayout.AddWidget(e.Separator, 0)
		e.BoxLayout.AddWidget(e.Editor, 0)
		e.EditorVisible = true
	}
}

// HideEditor makes the editor invisible and clears its content.
func (e *EphemeralEditor) HideEditor() {
	if e.EditorVisible {
		e.BoxLayout.RemoveWidget(e.Separator)
		e.BoxLayout.RemoveWidget(e.Editor)
		e.Editor.(*Editor).Clear()
		e.EditorVisible = false
	}
}

// HandleEvent processes events of interest to the EphemeralEditor.
// Notably, it handles EventSendRequest and EventReplyRequest by
// summoning and dismissing the editor widget.
func (e *EphemeralEditor) HandleEvent(ev tcell.Event) bool {
	switch event := ev.(type) {
	case EventReplyRequest:
		log.Printf("EphemeralEditor received event: %T %v", ev, ev)
		e.ShowEditor()
	case EventSendRequest:
		log.Printf("EphemeralEditor received event: %T %v", ev, ev)
		e.HideEditor()
	case views.EventWidget:
		e.PostEvent(event)
	case *tcell.EventKey:
		if e.EditorVisible {
			if e.Editor.HandleEvent(ev) {
				return true
			}
		}
		if e.PrimaryContent.HandleEvent(ev) {
			return true
		}
	}
	return false
}
