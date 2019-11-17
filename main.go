package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/0xAX/notificator"
	"github.com/gdamore/tcell/views"
	"github.com/pkg/profile"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
)

func CheckNotify() {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("notify-send"); err != nil {
			log.Println("WARNING: desktop notifications require `notify-send` to be installed")
		}
	}
}

func main() {
	CheckNotify()
	config := NewConfig()
	defer profile.Start(profile.ProfilePath(config.RuntimeDirectory)).Stop()

	if err := config.StartLogging(); err != nil {
		log.Fatalf("Failed to configure logging: %v", err)
	}

	flag.StringVar(&config.PGPUser, "gpguser", "", "gpg user to sign new messages with")
	flag.StringVar(&config.PGPKey, "key", "", "PGP key to sign messages with")
	var identityFile string
	flag.StringVar(&identityFile, "identity", "", "arbor identity node to sign with")
	flag.Usage = func() {
		executable := os.Args[0]
		fmt.Fprintf(flag.CommandLine.Output(), `Usage of %s:

%s [flags] [relay-address]

Where [relay-address] is the IP:PORT or FQDN:PORT of a sprout relay
and [flags] are among those listed below:

`, executable, executable)
		flag.PrintDefaults()
	}
	flag.Parse()

	b, err := ioutil.ReadFile(identityFile)
	if err != nil {
	}
	config.Identity, err = forest.UnmarshalIdentity(b)
	if err != nil {
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	store, err := grove.New(cwd)
	if err != nil {
		log.Fatalf("Failed to create grove at %s: %v", cwd, err)
	}
	if config.Validate() != nil {
		wizard := &Wizard{
			Config:   config,
			Prompter: &StdoutPrompter{In: os.Stdin, Out: os.Stdout},
		}
		if err := wizard.Run(store); err != nil {
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
