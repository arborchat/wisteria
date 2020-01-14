package widgets

import (
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// linesModel is modeled after the one in tcell's views package
type linesModel struct {
	lines  []string
	width  int
	height int
	x      int
	y      int
	hide   bool
	cursor bool
	style  tcell.Style
}

func (m *linesModel) GetCell(x, y int) (rune, tcell.Style, []rune, int) {
	var ch rune
	if x < 0 || y < 0 || y >= len(m.lines) || x >= len(m.lines[y]) {
		return ch, m.style, nil, 1
	}
	// XXX: extend this to support combining and full width chars
	return rune(m.lines[y][x]), m.style, nil, 1
}

func (m *linesModel) GetBounds() (int, int) {
	return m.width, m.height
}

func (m *linesModel) limitCursor() {
	if m.x > m.width-1 {
		m.x = m.width - 1
	}
	if m.y > m.height-1 {
		m.y = m.height - 1
	}
	if m.x < 0 {
		m.x = 0
	}
	if m.y < 0 {
		m.y = 0
	}
}

func (m *linesModel) SetCursor(x, y int) {
	m.x = x
	m.y = y
	m.limitCursor()
}

func (m *linesModel) MoveCursor(x, y int) {
	m.x += x
	m.y += y
	m.limitCursor()
}

func (m *linesModel) GetCursor() (int, int, bool, bool) {
	return m.x, m.y, m.cursor, !m.hide
}

// Writer is a TextArea modified to implement io.Writer
type Writer struct {
	once  sync.Once
	model *linesModel
	views.CellView
}

func NewWriterWidget() *Writer {
	w := &Writer{}
	w.Init()
	return w
}

// SetLines sets the content text to display.
func (ta *Writer) SetLines(lines []string) {
	ta.Init()
	m := ta.model
	m.width = 0
	m.height = len(lines)
	m.lines = append(m.lines, lines...)
	for _, l := range lines {
		if len(l) > m.width {
			m.width = len(l)
		}
	}
	x, y, _, _ := m.GetCursor()
	ta.CellView.SetModel(m)
	m.SetCursor(x, y)
}

func (ta *Writer) SetStyle(style tcell.Style) {
	ta.model.style = style
	ta.CellView.SetStyle(style)
}

// EnableCursor enables a soft cursor in the TextArea.
func (ta *Writer) EnableCursor(on bool) {
	ta.Init()
	ta.model.cursor = on
}

// HideCursor hides or shows the cursor in the TextArea.
// If on is true, the cursor is hidden.  Note that a cursor is only
// shown if it is enabled.
func (ta *Writer) HideCursor(on bool) {
	ta.Init()
	ta.model.hide = on
}

// Init initializes the TextArea.
func (ta *Writer) Init() {
	ta.once.Do(func() {
		lm := &linesModel{lines: []string{}, width: 0}
		ta.model = lm
		ta.CellView.Init()
		ta.CellView.SetModel(lm)
	})
}

// Write adds the given bytes to the end of the content in the
// TextArea
func (l *Writer) Write(b []byte) (int, error) {
	asString := string(b)
	lines := strings.Split(strings.TrimSpace(asString), "\n")
	l.SetLines(lines)
	l.PostEventWidgetContent(l)
	return len(b), nil
}
