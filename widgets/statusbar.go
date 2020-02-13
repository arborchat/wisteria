package widgets

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

type StatusBar struct {
	views.SimpleStyledTextBar
}

func NewStatusBar() *StatusBar {
	bar := &StatusBar{}
	bar.SetStyle(tcell.StyleDefault.Reverse(true))
	return bar
}

func (s *StatusBar) HandleEvent(ev tcell.Event) bool {
	switch event := ev.(type) {
	case EventReplySelected:
		s.SetLeft(fmt.Sprintf("%%SCommunity: %s", string(event.Community.Name.Blob)))
		timestamp := event.Selected.Created.Time().Local()
		s.SetCenter(fmt.Sprintf("%%SDepth: %d, Written: %s", event.Selected.Depth, timestamp.Format(time.Stamp)))
		s.SetRight(fmt.Sprintf("%%SID: %s", event.Selected.ID().String()[:20]))
		return true
	}
	return false
}
