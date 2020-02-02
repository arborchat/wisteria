package widgets

import (
	"time"

	"git.sr.ht/~whereswaldon/forest-go"
	"github.com/gdamore/tcell/views"
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
