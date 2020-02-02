package widgets

import (
	"time"

	"github.com/gdamore/tcell/views"
)

// EventEditRequest indicates that a widget has requested that an editor be
// presented to the user with the provided initial content. The ID field
// is provided by the creator of the request and should be included in any
// EventEditFinished event that is created as a result of this Request. This
// allows higher-level logic to correlate the Finished event with the Requst.
// It fulfills views.EventWidget.
type EventEditRequest struct {
	ID      int
	Content string
	At      time.Time
	On      views.Widget
}

// NewEventEditRequest creates a new request.
func NewEventEditRequest(id int, widget views.Widget, content string) EventEditRequest {
	return EventEditRequest{
		ID:      id,
		Content: content,
		At:      time.Now(),
		On:      widget,
	}
}

// When returns the time at which the event took place.
func (e EventEditRequest) When() time.Time {
	return e.At
}

// Widget returns the widget requesting the reply.
func (e EventEditRequest) Widget() views.Widget {
	return e.On
}

var _ views.EventWidget = EventEditRequest{}

// EventEditFinished indicates that an EventEditFinished has been processed by
// an editor and contains the final edited text that the user provided.
// It fulfills views.EventWidget.
type EventEditFinished struct {
	ID      int
	Content string
	At      time.Time
	On      views.Widget
}

// NewEventEditFinished creates a new finished event. The id provided should be an ID from
// the EventEditRequest that was just finished.
func NewEventEditFinished(id int, widget views.Widget, content string) EventEditFinished {
	return EventEditFinished{
		ID:      id,
		Content: content,
		At:      time.Now(),
		On:      widget,
	}
}

// When returns the time at which the event took place.
func (e EventEditFinished) When() time.Time {
	return e.At
}

// Widget returns the widget requesting the reply.
func (e EventEditFinished) Widget() views.Widget {
	return e.On
}

var _ views.EventWidget = EventEditFinished{}
