package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/wisteria/archive"
	"git.sr.ht/~whereswaldon/wisteria/widgets"
	wistTcell "git.sr.ht/~whereswaldon/wisteria/widgets/tcell"
	"github.com/0xAX/notificator"
	wrap "github.com/bbrks/wrap/v2"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// EditRequestMap tracks the outstanding edit requests (messages that
// are in the process of being replied to) and which nodes those replies
// are intended for. It provides concurrent-safe operations to get and ID
// for a replying to a particular node as well as for getting the node
// back from its ID.
type EditRequestMap struct {
	Requests map[int]forest.Node
	sync.Mutex
	current int
}

// NewEditRequestMap initializes a new map
func NewEditRequestMap() *EditRequestMap {
	return &EditRequestMap{
		Requests: make(map[int]forest.Node),
	}
}

// Insert chooses an ID for the node and stores the node with that ID.
// It returns the chosen ID.
func (e *EditRequestMap) Insert(node forest.Node) int {
	e.Lock()
	defer e.Unlock()
	i := e.current
	e.current++
	e.Requests[i] = node
	return i
}

// Delete retrieves the node associated with the given ID by removing
// it from the map.
func (e *EditRequestMap) Delete(id int) forest.Node {
	e.Lock()
	defer e.Unlock()
	node, present := e.Requests[id]
	if !present {
		return nil
	}
	delete(e.Requests, id)
	return node
}

// HistoryWidget is the controller for the chat history TUI
type HistoryWidget struct {
	*HistoryView
	*CellView
	*wistTcell.Application
	*forest.Builder
	*Config
	*notificator.Notificator
	*EditRequestMap
}

func NewHistoryWidget(app *wistTcell.Application, archive *archive.Archive, config *Config, notifier *notificator.Notificator) (*HistoryWidget, error) {
	hv := &HistoryView{
		Archive: archive,
	}
	if err := hv.Render(); err != nil {
		return nil, fmt.Errorf("failed initializing history view: %w", err)
	}
	cv := NewCellView()
	cv.SetModel(hv)
	cv.MakeCursorVisible()
	hv.SelectLastLine()

	builder, err := config.Builder(archive)
	if err != nil {
		return nil, fmt.Errorf("failed creating node builder for widget: %w", err)
	}
	return &HistoryWidget{
		HistoryView:    hv,
		CellView:       cv,
		Application:    app,
		Builder:        builder,
		Config:         config,
		Notificator:    notifier,
		EditRequestMap: NewEditRequestMap(),
	}, nil
}

var _ views.Widget = &HistoryWidget{}

func (v *HistoryWidget) ReadMessageFile(filename string) {
	v.Application.PostFunc(func() {
		log.Printf("Reading message from %s", filename)
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Printf("Failed reading %s: %v", filename, err)
			return
		}
		node, err := forest.UnmarshalBinaryNode(b)
		if err != nil {
			log.Printf("Failed parsing %s: %v", filename, err)
			return
		}
		err = v.Add(node)
		if err != nil {
			log.Printf("Failed adding %s: %v", filename, err)
			return
		}
		v.Sort()
		err = v.Render()
		if err != nil {
			log.Printf("Failed rendering %s: %v", filename, err)
			return
		}
		v.Application.Update()
		if reply, ok := node.(*forest.Reply); ok {
			v.TryNotify(reply)
		}
	})
}

// TryNotify checks whether a desktop notification should be sent
// and attempts to send it
func (v *HistoryWidget) TryNotify(reply *forest.Reply) {
	identity, err := v.Config.IdentityNode(v.Store)
	if err != nil {
		log.Printf("couldn't look up local identity: %v", err)
		return
	}
	username := strings.ToLower(string(identity.Name.Blob))
	messageText := strings.ToLower(string(reply.Content.Blob))
	if !strings.Contains(messageText, username) {
		return
	}
	author, has, err := v.Get(&reply.Author)
	if err != nil {
		log.Printf("Couldn't render desktop notification: %v", err)
		return
	} else if !has {
		log.Println("Couldn't render desktop notification: author information missing")
		return
	}
	log.Printf("Pushing notification: %v", v.Push("Arbor Mention from "+string(author.(*forest.Identity).Name.Blob), string(reply.Content.Blob), "", notificator.UR_NORMAL))
}

// StartReply begins a new reply with the currently-selected message as its
// parent.
func (v *HistoryWidget) StartReply() error {
	reply, msg, err := v.CurrentReplyConfig()
	if err != nil {
		return fmt.Errorf("failed getting current reply configuration: %w", err)
	}
	return v.StartNewNode(reply, msg)
}

func (v *HistoryWidget) CurrentReplyConfig() (forest.Node, string, error) {
	reply, err := v.CurrentReply()
	if err != nil {
		return nil, "", fmt.Errorf("couldn't determine current reply: %v", err)
	}
	msg := strings.Join(strings.Split(string(reply.Content.Blob), "\n"), "\n#")
	msg = fmt.Sprintf("# replying to %s\n", msg)
	return reply, msg, nil
}

func (v *HistoryWidget) NewConversationConfig() (forest.Node, string, error) {
	reply, err := v.CurrentReply()
	if err != nil {
		return nil, "", fmt.Errorf("couldn't determine current reply: %w", err)
	}
	community, _, err := v.GetCommunity(&reply.CommunityID)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't locate current community: %w", err)
	}
	msg := fmt.Sprintf("# starting new conversation in %s\n", string(community.(*forest.Community).Name.Blob))
	return community, msg, nil
}

func (v *HistoryWidget) EmitReplyRequest() error {
	reply, msg, err := v.CurrentReplyConfig()
	if err != nil {
		return fmt.Errorf("failed getting current reply configuration: %w", err)
	}
	v.EmitEditorRequest(reply, msg)
	return nil
}

func (v *HistoryWidget) EmitConversationRequest() error {
	community, msg, err := v.NewConversationConfig()
	if err != nil {
		return fmt.Errorf("failed getting current reply configuration: %w", err)
	}
	v.EmitEditorRequest(community, msg)
	return nil
}

func (v *HistoryWidget) EmitEditorRequest(parent forest.Node, startText string) {
	editReq := widgets.NewEventEditRequest(v.EditRequestMap.Insert(parent), v, startText)
	v.PostEvent(editReq)
}

// StartNewNode launches an Editor to write and send a new arbor node.
func (v *HistoryWidget) StartNewNode(parent forest.Node, startText string) error {
	file, err := ioutil.TempFile("", "arbor-msg")
	if err != nil {
		return fmt.Errorf("couldn't create temporary file for reply: %v", err)
	}
	// ensure this file descriptor is closed
	file.Close()
	// populate the file, but keep it closed
	err = ioutil.WriteFile(file.Name(), []byte(startText), 0660)
	if err != nil {
		return fmt.Errorf("couldn't write template into temporary file: %v", err)
	}
	editor := v.Config.EditFile(file.Name())
	if err := editor.Start(); err != nil {
		return fmt.Errorf("failed to start editor command: %v", err)
	}
	go v.FinishReply(parent, file.Name(), editor)
	return nil
}

// StartConversation begins a new conversation in the same community as the
// currently-selected message.
func (v *HistoryWidget) StartConversation() error {
	community, msg, err := v.NewConversationConfig()
	if err != nil {
		return fmt.Errorf("couldn't get current community config: %w", err)
	}
	return v.StartNewNode(community, msg)
}

// FinishReply waits for the provided editor command to complete (it is expected
// to have already started) and writes the contents of the named file as a new
// node.
func (v *HistoryWidget) FinishReply(parent forest.Node, replyFileName string, editor *exec.Cmd) {
	if err := editor.Wait(); err != nil {
		log.Printf("Error waiting on editor command to finish: %v", err)
		log.Printf("There may be a partial message in %s", replyFileName)
		return
	}
	replyContent, err := ioutil.ReadFile(replyFileName)
	if err != nil {
		log.Printf("Error reading reply from %s: %v", replyFileName, err)
		return
	}
	if err := v.FinishReplyString(parent, string(replyContent)); err != nil {
		log.Printf("Error creating & sending reply: %v", err)
		return
	}
	if err := os.Remove(replyFileName); err != nil {
		log.Printf("Error removing %s: %v", replyFileName, err)
		return
	}
}

// FinishReplyString writes the content provided as the content of a new forest
// node into the store.
func (v *HistoryWidget) FinishReplyString(parent forest.Node, content string) error {
	wrapper := wrap.NewWrapper()
	wrapper.Breakpoints = " "
	replyContentString := strings.Trim(wrapper.Wrap(stripCommentLines(content), 80), "\n")
	if len(replyContentString) == 0 {
		return fmt.Errorf("not sending empty message")
	}
	reply, err := v.NewReply(parent, replyContentString, []byte{})
	if err != nil {
		return fmt.Errorf("failed creating reply: %w", err)
	}

	err = v.Add(reply)
	if err != nil {
		return fmt.Errorf("failed saving reply into store: %w", err)
	}
	return nil
}

// UpdateCursor ensures that the cursor is visible and handles all necessary
// state changes each time the cursor moves. This includes firing events
// related to moving the cursor.
func (v *HistoryWidget) UpdateCursor() {
	v.MakeCursorVisible()
	current, err := v.CurrentReply()
	if err != nil {
		log.Printf("Failed updating cursor state: %v", err)
		return
	} else if current == nil {
		return
	}
	author, _, err := v.GetIdentity(&current.Author)
	if err != nil {
		log.Printf("Failed updating cursor state, couldn't get author: %v", err)
	}
	community, _, err := v.GetCommunity(&current.CommunityID)
	if err != nil {
		log.Printf("Failed updating cursor state, couldn't get community: %v", err)
	}
	v.PostEvent(widgets.NewEventReplySelected(v, current, author.(*forest.Identity), community.(*forest.Community)))
}

func (v *HistoryWidget) cursorToTop() {
	v.HistoryView.SetCursor(0, 0)
	v.UpdateCursor()
}

func (v *HistoryWidget) cursorToBottom() {
	v.SelectLastLine()
	v.UpdateCursor()
}

func (v *HistoryWidget) panUp(rows int) {
	v.CellView.PanUp(rows)
}

func (v *HistoryWidget) cursorUpOneLine() {
	v.MoveCursor(0, -1)
	v.UpdateCursor()
}

func (v *HistoryWidget) panDown(rows int) {
	v.CellView.PanDown(rows)
}

func (v *HistoryWidget) cursorDownOneLine() {
	v.MoveCursor(0, 1)
	v.UpdateCursor()
}

func (v *HistoryWidget) panLeft(cols int) {
	v.CellView.PanLeft(cols)
}

func (v *HistoryWidget) cursorLeftOneCell() {
	v.MoveCursor(-1, 0)
	v.UpdateCursor()
}

func (v *HistoryWidget) panRight(cols int) {
	v.CellView.PanRight(cols)
}

func (v *HistoryWidget) cursorRightOneCell() {
	v.MoveCursor(1, 0)
	v.UpdateCursor()
}

func (v *HistoryWidget) HandleEvent(event tcell.Event) bool {
	const mouseScrollMultiplier = 5
	if v.CellView.HandleEvent(event) {
		return true
	}
	switch keyEvent := event.(type) {
	case widgets.EventEditFinished:
		log.Printf("Got event edit finished: %v", keyEvent)
		if err := v.FinishReplyString(v.EditRequestMap.Delete(keyEvent.ID), keyEvent.Content); err != nil {
			log.Printf("Failed finalizing reply: %v", err)
		}
	case *tcell.EventMouse:
		buttons := keyEvent.Buttons()
		switch {
		case buttons&tcell.Button1 > 0:
			// TODO(whereswaldon): avoid hard-coding the height of the top
			// status bar here. This requires teaching the views package
			// how to translate these clicks coordinates between the coordinate
			// systems of the physical terminal and each widget, which it does
			// not currently support.
			const TopBarHeight = 1
			const LeftContentWidth = 0
			physicalX, physicalY := keyEvent.Position()
			log.Println("Detected click:", physicalX, physicalY)
			ulVizX, ulVizY, _, _ := v.port.GetVisible()
			log.Println("UL Physical:", ulVizX, ulVizY)
			logicalX, logicalY := physicalX+ulVizX-LeftContentWidth, physicalY+ulVizY-TopBarHeight
			log.Println("Logical:", logicalX, logicalY)
			v.model.SetCursor(logicalX, logicalY)
		case buttons&tcell.WheelUp > 0:
			v.panUp(mouseScrollMultiplier)
		case buttons&tcell.WheelDown > 0:
			v.panDown(mouseScrollMultiplier)
		case buttons&tcell.WheelLeft > 0:
			v.panLeft(mouseScrollMultiplier)
		case buttons&tcell.WheelRight > 0:
			v.panRight(mouseScrollMultiplier)
		}
	case *tcell.EventKey:
		switch keyEvent.Key() {
		case tcell.KeyEnter:
			if err := v.EmitReplyRequest(); err != nil {
				log.Printf("Error starting reply: %v", err)
				return true
			}
		case tcell.KeyUp, tcell.KeyCtrlP:
			v.cursorUpOneLine()
			return true
		case tcell.KeyDown, tcell.KeyCtrlN:
			v.cursorDownOneLine()
			return true
		case tcell.KeyRight, tcell.KeyCtrlF:
			v.cursorRightOneCell()
			return true
		case tcell.KeyLeft, tcell.KeyCtrlB:
			v.cursorLeftOneCell()
			return true
		case tcell.KeyPgDn:
			v.keyPgDn()
			v.UpdateCursor()
			return true
		case tcell.KeyPgUp:
			v.keyPgUp()
			v.UpdateCursor()
			return true
		case tcell.KeyEnd:
			v.cursorToBottom()
			return true
		case tcell.KeyHome:
			v.cursorToTop()
			return true
		case tcell.KeyRune:
			// break if it's a normal keypress
		default:
			return false
		}
		switch keyEvent.Rune() {
		case 'g':
			v.cursorToTop()
			return true
		case 'G':
			v.cursorToBottom()
			return true
		case 'h':
			v.cursorLeftOneCell()
			return true
		case 'j':
			v.cursorDownOneLine()
			return true
		case 'k':
			v.cursorUpOneLine()
			return true
		case 'l':
			v.cursorRightOneCell()
			return true
		case 'c':
			if err := v.EmitConversationRequest(); err != nil {
				log.Printf("Error starting conversation: %v", err)
				return true
			}
		case 'C':
			if err := v.StartConversation(); err != nil {
				log.Printf("Error starting conversation: %v", err)
				return true
			}
		case 'i':
			if err := v.EmitReplyRequest(); err != nil {
				log.Printf("Error starting conversation: %v", err)
				return true
			}
		case 'I':
			if err := v.StartReply(); err != nil {
				log.Printf("Error starting conversation: %v", err)
				return true
			}
		case ' ':
			v.ToggleFilter()
			if err := v.Render(); err != nil {
				log.Printf("Error re-rendering after filter: %v", err)
			}
			v.Draw()
			x, y, _, _ := v.GetCursor()
			v.port.Center(x, y)
			return true
		}
	}
	return false
}
