package widgets

import (
	"time"

	"git.sr.ht/~whereswaldon/forest-go"
	"github.com/gdamore/tcell/views"
)

// BasicEvent is an embeddable type that implements the views.EventWidget
// interface.
type BasicEvent struct {
	At time.Time
	On views.Widget
}

func NewBasicEvent(widget views.Widget) BasicEvent {
	return BasicEvent{
		At: time.Now(),
		On: widget,
	}
}

// When returns the time at which the event took place.
func (e BasicEvent) When() time.Time {
	return e.At
}

// Widget returns the widget requesting the reply.
func (e BasicEvent) Widget() views.Widget {
	return e.On
}

// EventReplySelected indicates that a particular node was selected in the TUI.
type EventReplySelected struct {
	Selected  *forest.Reply
	Author    *forest.Identity
	Community *forest.Community
	BasicEvent
}

// NewEventReplySelected creates an event indicating which node was selected as
// well as providing a copy of its author and parent community nodes.
func NewEventReplySelected(widget views.Widget, selected *forest.Reply, author *forest.Identity, community *forest.Community) EventReplySelected {
	return EventReplySelected{
		BasicEvent: NewBasicEvent(widget),
		Selected:   selected,
		Author:     author,
		Community:  community,
	}
}

// EventEditRequest indicates that a widget has requested that an editor be
// presented to the user with the provided initial content. The ID field
// is provided by the creator of the request and should be included in any
// EventEditFinished event that is created as a result of this Request. This
// allows higher-level logic to correlate the Finished event with the Requst.
// It fulfills views.EventWidget.
type EventEditRequest struct {
	ID      int
	Content string
	BasicEvent
}

// NewEventEditRequest creates a new request.
func NewEventEditRequest(id int, widget views.Widget, content string) EventEditRequest {
	return EventEditRequest{
		ID:         id,
		Content:    content,
		BasicEvent: NewBasicEvent(widget),
	}
}

var _ views.EventWidget = EventEditRequest{}

// EventEditFinished indicates that an EventEditFinished has been processed by
// an editor and contains the final edited text that the user provided.
// It fulfills views.EventWidget.
type EventEditFinished struct {
	ID      int
	Content string
	BasicEvent
}

// NewEventEditFinished creates a new finished event. The id provided should be an ID from
// the EventEditRequest that was just finished.
func NewEventEditFinished(id int, widget views.Widget, content string) EventEditFinished {
	return EventEditFinished{
		ID:         id,
		Content:    content,
		BasicEvent: NewBasicEvent(widget),
	}
}

var _ views.EventWidget = EventEditFinished{}
