package widgets

import (
	"log"
	"time"

	"git.sr.ht/~whereswaldon/forest-go"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"github.com/mattn/go-runewidth"
)

// EventReplyRequest indicates that a widget has requested the composition
// of a reply to a specific node. It fulfills views.EventWidget.
type EventReplyRequest struct {
	ReplyingTo forest.Node
	At         time.Time
	On         views.Widget
}

func NewEventReplyRequest(widget views.Widget, node forest.Node) EventReplyRequest {
	return EventReplyRequest{
		ReplyingTo: node,
		At:         time.Now(),
		On:         widget,
	}
}

// When returns the time at which the event took place.
func (e EventReplyRequest) When() time.Time {
	return e.At
}

// Widget returns the widget requesting the reply.
func (e EventReplyRequest) Widget() views.Widget {
	return e.On
}

var _ views.EventWidget = EventReplyRequest{}

// EventSendRequest indicates that a widget has requested the sending
// of an embedded reply node. It fulfills views.EventWidget.
type EventSendRequest struct {
	Reply forest.Node
	At    time.Time
	On    views.Widget
}

func NewEventSendRequest(widget views.Widget, node forest.Node) EventSendRequest {
	return EventSendRequest{
		Reply: node,
		At:    time.Now(),
		On:    widget,
	}
}

// When returns the time at which the event took place.
func (e EventSendRequest) When() time.Time {
	return e.At
}

// Widget returns the widget requesting the reply.
func (e EventSendRequest) Widget() views.Widget {
	return e.On
}

var _ views.EventWidget = EventSendRequest{}

type Editor struct {
	*views.TextArea
	content string
}

func NewEditor() *Editor {
	e := &Editor{
		TextArea: views.NewTextArea(),
	}
	e.TextArea.EnableCursor(true)
	e.UpdateContent()
	return e
}

func (e *Editor) HandleEvent(ev tcell.Event) bool {
	switch event := ev.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyEnter:
			e.PostEvent(NewEventSendRequest(e, nil))
			return true
		case tcell.KeyBackspace:
			fallthrough
		case tcell.KeyBackspace2:
			e.UntypeRune()
			return true
		case tcell.KeyRune:
			e.TypeRune(event.Rune())
			return true
		default:
			// suppress cursor manipulation since we can't use it
			//			return e.TextArea.HandleEvent(ev)
		}
	}
	return false
}

func (e *Editor) TypeRune(keypress rune) {
	e.content += string(keypress)
	e.UpdateContent()
}

func (e *Editor) UntypeRune() {
	if len(e.content) < 1 {
		return
	}
	asRunes := []rune(e.content)
	e.content = string(asRunes[:len(asRunes)-1])
	e.UpdateContent()
}

func (e *Editor) UpdateContent() {
	width := runewidth.StringWidth(e.content)
	// add empty space so that the cursor has somewhere to be
	e.TextArea.SetContent(e.content + " ")
	e.TextArea.SetCursorX(width)
}

func (e *Editor) Clear() {
	e.content = ""
	e.UpdateContent()
}

type EphemeralEditor struct {
	PrimaryContent    views.Widget
	Separator, Editor views.Widget
	EditorVisible     bool
	*views.BoxLayout
	views.WidgetWatchers
}

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

func (e *EphemeralEditor) ShowEditor() {
	e.BoxLayout.AddWidget(e.Separator, 0)
	e.BoxLayout.AddWidget(e.Editor, 0)
	e.EditorVisible = true
}

func (e *EphemeralEditor) HideEditor() {
	e.BoxLayout.RemoveWidget(e.Separator)
	e.BoxLayout.RemoveWidget(e.Editor)
	e.Editor.(*Editor).Clear()
	e.EditorVisible = false
}

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
