package widgets

import (
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"github.com/mattn/go-runewidth"
)

// Editor implements a simple text editor as a widget. It emits
// EventSendRequest when an edited message is ready to be sent.
type Editor struct {
	*views.TextArea
	content string
}

// NewEditor constructs an empty Editor()
func NewEditor() *Editor {
	e := &Editor{
		TextArea: views.NewTextArea(),
	}
	e.TextArea.EnableCursor(true)
	e.UpdateContent()
	return e
}

// HandleEvent handles keypresses.
func (e *Editor) HandleEvent(ev tcell.Event) bool {
	switch event := ev.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyEnter:
			// don't provide an ID because we don't actually know the right one
			// higher level logic should populate it.
			e.PostEvent(NewEventEditFinished(0, e, e.content))
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

// Type rune adds the provided rune to the content of the editor.
func (e *Editor) TypeRune(keypress rune) {
	e.content += string(keypress)
	e.UpdateContent()
}

// Untype rune deletes the last rune in the editor.
func (e *Editor) UntypeRune() {
	if len(e.content) < 1 {
		return
	}
	asRunes := []rune(e.content)
	e.content = string(asRunes[:len(asRunes)-1])
	e.UpdateContent()
}

// UpdateContent synchronizes the internal editor state and the visible
// editor state.
func (e *Editor) UpdateContent() {
	width := runewidth.StringWidth(e.content)
	// add empty space so that the cursor has somewhere to be
	e.TextArea.SetContent(e.content + " ")
	e.TextArea.SetCursorX(width)
}

// Clear erases the content of the editor.
func (e *Editor) Clear() {
	e.content = ""
	e.UpdateContent()
}
