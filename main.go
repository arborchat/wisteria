package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/0xAX/notificator"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"github.com/pkg/profile"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/sprout-go"
	"git.sr.ht/~whereswaldon/sprout-go/watch"
	"git.sr.ht/~whereswaldon/wisteria/archive"
)

func CheckNotify() {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("notify-send"); err != nil {
			log.Println("WARNING: desktop notifications require `notify-send` to be installed")
		}
	}
}

func main() {
	// configure our usage information
	flag.Usage = func() {
		executable := os.Args[0]
		fmt.Fprintf(flag.CommandLine.Output(), `Usage of %s:

%s [flags] [relay-address [relay-address]...]

Where [relay-address] is the IP:PORT or FQDN:PORT of a sprout relay
and [flags] are among those listed below:

`, executable, executable)
		flag.PrintDefaults()
	}
	flag.Parse()

	// check whether we can send desktop notifications and warn if we can't
	CheckNotify()

	// make basic configuration
	config := NewConfig()

	// profile to runtime directory chosen by config
	defer profile.Start(profile.ProfilePath(config.RuntimeDirectory)).Stop()

	// set up logging to runtime directory
	if err := config.StartLogging(); err != nil {
		log.Fatalf("Failed to configure logging: %v", err)
	}

	// look up our working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// use a grove rooted in our current working directory as our node storage
	groveStore, err := grove.New(cwd)
	if err != nil {
		log.Fatalf("Failed to create grove at %s: %v", cwd, err)
	}

	// Wrap store in CacheStore
	cache := forest.NewMemoryStore()
	store, err := forest.NewCacheStore(cache, groveStore)
	if err != nil {
		log.Fatal("Failed to wrap Store in CacheStore:", err)
	}

	if err := config.LoadFromDefault(); err != nil {
		log.Printf("Failed loading configuration file: %v", err)
		// ask user for interactive configuration
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
		if err := config.SaveToDefault(); err != nil {
			log.Fatal("Error saving configuration:", err)
		}
	}

	// get a node builder from config so we can sign nodes
	builder, err := config.Builder(store)
	if err != nil {
		log.Fatal("Unable to construct builder using configuration:", err)
	}

	// create the observable message storage abstraction that sprout workers use
	subscriberStore := sprout.NewSubscriberStore(store)
	// create the queryable store abstraction that we need
	history, err := archive.NewArchive(subscriberStore)
	if err != nil {
		log.Fatalf("Failed to create archive: %v", err)
	}

	// dial relay address (if provided)
	done := make(chan struct{})
	for _, address := range flag.Args() {
		sprout.LaunchSupervisedWorker(done, address, subscriberStore, nil, log.New(log.Writer(), "", log.Flags()))
	}

	// ensure its internal state is what we want
	history.Sort()

	// make a TUI view of that history
	historyView := &HistoryView{
		Archive: history,
	}
	if err := historyView.Render(); err != nil {
		log.Fatal(err)
	}

	// wrap TUI view in the necessary tcell abstractions
	cv := NewCellView()
	cv.SetModel(historyView)
	cv.MakeCursorVisible()
	historyView.SelectLastLine() // start at bottom of history

	// set up desktop notifications
	notify := notificator.New(notificator.Options{
		AppName: "Arbor",
	})

	// build an widget/application from existing views and services
	app := new(views.Application)
	hw := &HistoryWidget{
		historyView,
		cv,
		app,
		builder,
		config,
		notify,
	}

	titlebar := views.NewSimpleStyledTextBar()
	titlebar.SetLeft("%Swisteria")
	titlebar.SetRight("%Sarrows or vi to move; enter to reply; c for new convo")
	titlebar.SetStyle(tcell.StyleDefault.Reverse(true))

	layout := views.NewBoxLayout(views.Vertical)
	layout.AddWidget(titlebar, 0)
	layout.AddWidget(hw, 1)
	app.SetRootWidget(layout)

	// watch the cwd for new nodes from other sources
	logger := log.New(log.Writer(), "", log.LstdFlags|log.Lshortfile)
	if _, err := watch.Watch(cwd, logger, hw.ReadMessageFile); err != nil {
		log.Fatal(err)
	}

	// run the TUI
	if e := app.Run(); e != nil {
		log.Println(e.Error())
		os.Exit(1)
	}
}
