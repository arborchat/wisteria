package widgets

import (
	"log"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// EphemeralEditor is a layout that can summon and dismiss an Editor
// widget as needed. When it receives EventEditRequest, it will create
// an Editor widget that will accept all key events before its PrimaryContent.
// Once that Editor emits EventEditFinished, it will dismiss the editor.
type EphemeralEditor struct {
	PrimaryContent               views.Widget
	Separator, Editor, Requestor views.Widget
	// the ID field of the EventEditRequest currently being handled, if any
	RequestID     int
	EditorVisible bool
	*views.BoxLayout
	views.WidgetWatchers
}

// NewEphemeralEditor creates a new layout with the given view as the
// primary content.
func NewEphemeralEditor(primary views.Widget) *EphemeralEditor {
	separator := views.NewTextBar()
	style := tcell.StyleDefault.Reverse(true)
	separator.SetStyle(style)
	separator.SetLeft("Type your reply below; Enter to send; Send empty message to cancel", style)
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

func (e *EphemeralEditor) SetRequestor(event EventEditRequest) {
	if e.Requestor == nil {
		e.Requestor = event.On
		e.RequestID = event.ID
	} else {
		log.Printf("Warning: Tried to set edit requestor while one was already set.")
	}
}

func (e *EphemeralEditor) ClearRequestor() {
	if e.Requestor != nil {
		e.Requestor = nil
		e.RequestID = 0
	} else {
		log.Printf("Warning: Tried to clear edit requestor while none was set.")
	}
}

// HandleEvent processes events of interest to the EphemeralEditor.
// Notably, it handles EventEditRequest and EventEditFinished by
// summoning and dismissing the editor widget.
func (e *EphemeralEditor) HandleEvent(ev tcell.Event) bool {
	switch event := ev.(type) {
	case EventEditRequest:
		e.ShowEditor()
		e.SetRequestor(event)
		return true
	case EventEditFinished:
		// pass the EventEditFinished to the widget that created the
		// EventEditRequest with the ID field populated
		event.ID = e.RequestID
		defer func() {
			// clean up after handling event
			e.HideEditor()
			e.ClearRequestor()
		}()
		return e.Requestor.HandleEvent(event)
	case views.EventWidget:
		e.PostEvent(event)
	case *tcell.EventMouse:
		return e.passEventDown(ev)
	case *tcell.EventKey:
		return e.passEventDown(ev)
	}
	return false
}

// passEventDown sends the input event to the editor if it is visible and
// otherwise to the underlying view.
func (e *EphemeralEditor) passEventDown(ev tcell.Event) bool {
	if e.EditorVisible {
		if e.Editor.HandleEvent(ev) {
			return true
		}
	}
	if e.PrimaryContent.HandleEvent(ev) {
		return true
	}
	return false
}
