package main

import (
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// SwitcherLayout extends BoxLayout with simple facilities to switch out
// the primary content widget in the BoxLayout.
type SwitcherLayout struct {
	*views.Application
	LogWidget     views.Widget
	ContentWidget views.Widget

	Current views.Widget

	views.WidgetWatchers
}

// NewSwitcherLayout creates a SwitcherLayout with the given views as its
// Content and Log widgets.
func NewSwitcherLayout(app *views.Application, content, log views.Widget) *SwitcherLayout {
	s := &SwitcherLayout{
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

func (s *SwitcherLayout) Draw() {
	s.Current.Draw()
}

func (s *SwitcherLayout) Resize() {
	s.ContentWidget.Resize()
	s.LogWidget.Resize()
}

func (s *SwitcherLayout) SetView(view views.View) {
	s.ContentWidget.SetView(view)
	s.LogWidget.SetView(view)
}

func (s *SwitcherLayout) Size() (int, int) {
	return s.Current.Size()
}

func (s *SwitcherLayout) HandleEvent(ev tcell.Event) bool {
	if s.Current.HandleEvent(ev) {
		return true
	}
	switch keyEvent := ev.(type) {
	case *views.EventWidgetContent:
		// propagate content events upward
		s.Application.Update()
	case *tcell.EventKey:
		switch keyEvent.Key() {
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

func (s *SwitcherLayout) ToggleLogWidget() {
	if s.Current == s.LogWidget {
		s.Current = s.ContentWidget
		return
	}
	s.Current = s.LogWidget
}
