package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"github.com/0xAX/notificator"
	wrap "github.com/bbrks/wrap/v2"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

// HistoryWidget is the controller for the chat history TUI
type HistoryWidget struct {
	*HistoryView
	*CellView
	*views.Application
	*forest.Builder
	*Config
	*notificator.Notificator
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
	username := strings.ToLower(string(v.Config.Identity.Name.Blob))
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
	reply, err := v.CurrentReply()
	if err != nil {
		return fmt.Errorf("couldn't determine current reply: %v", err)
	}
	msg := strings.Join(strings.Split(string(reply.Content.Blob), "\n"), "\n#")
	file, err := ioutil.TempFile("", "arbor-msg")
	if err != nil {
		return fmt.Errorf("couldn't create temporary file for reply: %v", err)
	}
	// ensure this file descriptor is closed
	file.Close()
	// populate the file, but keep it closed
	err = ioutil.WriteFile(file.Name(), []byte(fmt.Sprintf("# replying to %s\n", msg)), 0660)
	if err != nil {
		return fmt.Errorf("couldn't write template into temporary file: %v", err)
	}
	editor := v.Config.EditFile(file.Name())
	if err := editor.Start(); err != nil {
		return fmt.Errorf("failed to start editor command: %v", err)
	}
	go v.FinishReply(reply, file.Name(), editor)
	return nil
}

// StartConversation begins a new conversation in the same community as the
// currently-selected message.
func (v *HistoryWidget) StartConversation() error {
	reply, err := v.CurrentReply()
	if err != nil {
		return fmt.Errorf("couldn't determine current reply: %w", err)
	}
	community, _, err := v.GetCommunity(&reply.CommunityID)
	if err != nil {
		return fmt.Errorf("couldn't locate current community: %w", err)
	}
	file, err := ioutil.TempFile("", "arbor-msg")
	if err != nil {
		return fmt.Errorf("couldn't create temporary file for reply: %v", err)
	}
	// ensure this file descriptor is closed
	file.Close()
	// populate the file, but keep it closed
	err = ioutil.WriteFile(file.Name(), []byte(fmt.Sprintf("# starting new conversation in %s\n", string(community.(*forest.Community).Name.Blob))), 0660)
	if err != nil {
		return fmt.Errorf("couldn't write template into temporary file: %v", err)
	}
	editor := v.Config.EditFile(file.Name())
	if err := editor.Start(); err != nil {
		return fmt.Errorf("failed to start editor command: %v", err)
	}
	go v.FinishReply(community, file.Name(), editor)
	return nil
}

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
	replyContentString := strings.Trim(wrap.Wrap(stripCommentLines(string(replyContent)), 80), "\n")
	if len(replyContentString) == 0 {
		log.Println("Message is empty, not sending")
		return
	}
	reply, err := v.NewReply(parent, replyContentString, "")
	if err != nil {
		log.Printf("Error creating reply: %v", err)
		return
	}
	outfile, err := reply.ID().MarshalString()
	if err != nil {
		log.Printf("Error finding ID for reply: %v", err)
		return
	}
	err = saveAs(outfile, reply)
	if err != nil {
		log.Printf("Error saving to %s: %v", outfile, err)
		return
	}
	if err := os.Remove(replyFileName); err != nil {
		log.Printf("Error removing %s: %v", replyFileName, err)
		return
	}
}

func (v *HistoryWidget) HandleEvent(event tcell.Event) bool {
	if v.CellView.HandleEvent(event) {
		return true
	}
	switch keyEvent := event.(type) {
	case *tcell.EventKey:
		switch keyEvent.Key() {
		case tcell.KeyCtrlC:
			v.Application.Quit()
		case tcell.KeyEnter:
			if err := v.StartReply(); err != nil {
				log.Printf("Error starting reply: %v", err)
				return true
			}
		case tcell.KeyRune:
			// break if it's a normal keypress
		default:
			return false
		}
		switch keyEvent.Rune() {
		case 'g':
			v.HistoryView.SetCursor(0, 0)
			v.MakeCursorVisible()
			return true
		case 'G':
			v.SelectLastLine()
			v.MakeCursorVisible()
			return true
		case 'h':
			v.MoveCursor(-1, 0)
			v.MakeCursorVisible()
			return true
		case 'j':
			v.MoveCursor(0, 1)
			v.MakeCursorVisible()
			return true
		case 'k':
			v.MoveCursor(0, -1)
			v.MakeCursorVisible()
			return true
		case 'l':
			v.MoveCursor(1, 0)
			v.MakeCursorVisible()
			return true
		case 'c':
			if err := v.StartConversation(); err != nil {
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
