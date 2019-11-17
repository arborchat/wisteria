package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/0xAX/notificator"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"github.com/pkg/profile"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
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
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Println(err)
			return
		}
		node, err := forest.UnmarshalBinaryNode(b)
		if err != nil {
			log.Println(err)
			return
		}
		err = v.Add(node)
		if err != nil {
			log.Println(err)
			return
		}
		v.Sort()
		err = v.Render()
		if err != nil {
			log.Println(err)
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

func (v *HistoryWidget) FinishReply(parent *forest.Reply, replyFileName string, editor *exec.Cmd) {
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
	replyContentString := strings.Trim(stripCommentLines(string(replyContent)), "\n")
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
		}
	}
	return false
}

func CheckNotify() {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("notify-send"); err != nil {
			log.Println("WARNING: desktop notifications require `notify-send` to be installed")
		}
	}
}

func main() {
	CheckNotify()
	var identityFile string
	config := NewConfig()
	defer profile.Start(profile.ProfilePath(config.RuntimeDirectory)).Stop()
	logPath := path.Join(config.RuntimeDirectory, "viewer.log")
	log.Println("Logging to", logPath)
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	flag.StringVar(&config.PGPUser, "gpguser", "", "gpg user to sign new messages with")
	flag.StringVar(&config.PGPKey, "key", "", "PGP key to sign messages with")
	flag.StringVar(&identityFile, "identity", "", "arbor identity node to sign with")
	flag.Parse()
	b, err := ioutil.ReadFile(identityFile)
	if err != nil {
	}
	config.Identity, err = forest.UnmarshalIdentity(b)
	if err != nil {
	}
	if flag.NArg() > 0 {
		config.EditorCmd = flag.Args()
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if config.Validate() != nil {
		wizard := &Wizard{
			Config:   config,
			Prompter: &StdoutPrompter{In: os.Stdin, Out: os.Stdout},
		}
		if err := wizard.Run(cwd); err != nil {
			log.Fatal("Error running configuration wizard:", err)
		}
		if err := config.Validate(); err != nil {
			log.Fatal("Error validating configuration:", err)
		}
	}
	builder, err := config.Builder()
	if err != nil {
		log.Fatal("Unable to construct builder using configuration:", err)
	}
	store, err := grove.New(cwd)
	if err != nil {
		log.Fatalf("Failed to create grove at %s: %v", cwd, err)
	}
	history, err := NewArchive(store)
	if err != nil {
		log.Fatalf("Failed to create archive: %v", err)
	}
	history.Sort()
	historyView := &HistoryView{
		Archive: history,
	}
	if err := historyView.Render(); err != nil {
		log.Fatal(err)
	}
	cv := NewCellView()
	cv.SetModel(historyView)
	cv.MakeCursorVisible()
	historyView.SelectLastLine() // start at bottom of history

	// set up desktop notifications
	notify := notificator.New(notificator.Options{
		AppName: "Arbor",
	})

	app := new(views.Application)
	hw := &HistoryWidget{
		historyView,
		cv,
		app,
		builder,
		config,
		notify,
	}
	app.SetRootWidget(hw)

	if _, err := Watch(cwd, hw.ReadMessageFile); err != nil {
		log.Fatal(err)
	} else {
		//		defer watcher.Close()
	}

	if e := app.Run(); e != nil {
		log.Println(e.Error())
		os.Exit(1)
	}
}
