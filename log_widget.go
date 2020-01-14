package main

import (
	"strings"

	"github.com/gdamore/tcell/views"
)

// WriterWidget is a TextArea modified to implement io.Writer
type WriterWidget struct {
	*views.TextArea
	text []string
}

func NewWriterWidget() *WriterWidget {
	w := &WriterWidget{
		TextArea: views.NewTextArea(),
		text:     nil,
	}
	w.TextArea.Init()
	return w
}

// Write adds the given bytes to the end of the content in the
// TextArea
func (l *WriterWidget) Write(b []byte) (int, error) {
	asString := string(b)
	lines := strings.Split(strings.TrimSpace(asString), "\n")
	l.text = append(l.text, lines...)
	l.TextArea.SetLines(l.text)
	l.PostEventWidgetContent(l)
	return len(b), nil
}
