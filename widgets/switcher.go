package widgets

import (
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// Switcher allows toggling a single widget between multiple underlying widgets
// with only one widget visible at a time. Only the visible widget receives
// events.
type Switcher struct {
	*views.Application
	LogWidget     views.Widget
	ContentWidget views.Widget

	Current views.Widget

	views.WidgetWatchers
}

// NewSwitcher creates a Switcher with the given views as its
// Content and Log widgets.
func NewSwitcher(app *views.Application, content, log views.Widget) *Switcher {
	s := &Switcher{
		Application:   app,
		ContentWidget: content,
		LogWidget:     log,
	}
	s.Current = s.ContentWidget

	// subscribe to the events of child widgets
	log.Watch(s)
	content.Watch(s)
	return s
}

func (s *Switcher) Draw() {
	s.Current.Draw()
}

func (s *Switcher) Resize() {
	s.ContentWidget.Resize()
	s.LogWidget.Resize()
}

func (s *Switcher) SetView(view views.View) {
	s.ContentWidget.SetView(view)
	s.LogWidget.SetView(view)
}

func (s *Switcher) Size() (int, int) {
	return s.Current.Size()
}

func (s *Switcher) HandleEvent(ev tcell.Event) bool {
	if s.Current.HandleEvent(ev) {
		return true
	}
	switch keyEvent := ev.(type) {
	case *views.EventWidgetContent:
		// propagate content events upward
		s.Application.Update()
	case *tcell.EventKey:
		switch keyEvent.Key() {
		case tcell.KeyCtrlC:
			s.Application.Quit()
		case tcell.KeyRune:
			switch keyEvent.Rune() {
			case 'L':
				s.ToggleLogWidget()
				return true
			}
		}
	}
	return false
}

func (s *Switcher) ToggleLogWidget() {
	if s.Current == s.LogWidget {
		s.Current = s.ContentWidget
		return
	}
	s.Current = s.LogWidget
}