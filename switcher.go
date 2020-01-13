package main

import (
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// SwitcherLayout extends BoxLayout with simple facilities to switch out
// the primary content widget in the BoxLayout.
type SwitcherLayout struct {
	*views.BoxLayout
	LogWidget     views.Widget
	ContentWidget views.Widget
}

const primaryViewIndex = 0

// NewSwitcherLayout creates a SwitcherLayout with the given views as its
// Content and Log widgets.
func NewSwitcherLayout(content, log views.Widget) *SwitcherLayout {
	s := &SwitcherLayout{
		ContentWidget: content,
		LogWidget:     log,
		BoxLayout:     views.NewBoxLayout(views.Vertical),
	}
	s.BoxLayout.InsertWidget(primaryViewIndex, content, 1)
	return s
}

func (s *SwitcherLayout) HandleEvent(ev tcell.Event) bool {
	if s.BoxLayout.HandleEvent(ev) {
		return true
	}
	switch keyEvent := ev.(type) {
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
	if s.Widgets()[primaryViewIndex] == s.LogWidget {
		s.RemoveWidget(s.LogWidget)
		s.InsertWidget(primaryViewIndex, s.ContentWidget, 1)
		return
	}
	s.RemoveWidget(s.ContentWidget)
	s.InsertWidget(primaryViewIndex, s.LogWidget, 1)
}
