package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/0xAX/notificator"
	"github.com/gdamore/tcell/views"
	"github.com/pkg/profile"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/sprout-go"
	"git.sr.ht/~whereswaldon/wisteria/watch"
)

func CheckNotify() {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("notify-send"); err != nil {
			log.Println("WARNING: desktop notifications require `notify-send` to be installed")
		}
	}
}

func LaunchWorker(address string, store forest.Store) (*sprout.Worker, error) {
	doneChan := make(chan struct{})
	tcpConn, err := tls.Dial("tcp", address, &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed dialing %s: %w", address, err)
	}
	substore := sprout.NewSubscriberStore(store)
	worker, err := sprout.NewWorker(doneChan, tcpConn, substore)
	if err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("failed starting sprout worker: %v", err)
	}
	go worker.Run()
	return worker, nil
}

func main() {
	// configure our usage information
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
	store, err := grove.New(cwd)
	if err != nil {
		log.Fatalf("Failed to create grove at %s: %v", cwd, err)
	}

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

	// get a node builder from config so we can sign nodes
	builder, err := config.Builder()
	if err != nil {
		log.Fatal("Unable to construct builder using configuration:", err)
	}

	// create the queryable store abstraction that we need
	history, err := NewArchive(store)
	if err != nil {
		log.Fatalf("Failed to create archive: %v", err)
	}

	// dial relay address (if provided)
	if flag.NArg() > 0 {
		address := flag.Arg(0)
		worker, err := LaunchWorker(address, store)
		if err != nil {
			log.Printf("Failed to launch worker: %v", err)
		} else {
			log.Printf("Launched sprout worker connected to %s", address)
		}
		_, err = worker.SendList(fields.NodeTypeCommunity, 1024)
		if err != nil {
			log.Printf("Failed sending list verb to fetch communities: %v", err)
		}
		time.Sleep(time.Second)
		communities, err := store.Recent(fields.NodeTypeCommunity, 1024)
		if err != nil {
			log.Printf("Failed loading known recent communities: %v", err)
		}
		for _, community := range communities {
			if _, err := worker.SendSubscribe(community.(*forest.Community)); err != nil {
				log.Printf("Failed subscribing to community %s: %v", community.ID().String(), err)
			} else {
				log.Printf("Subscribed to %s", community.ID().String())
			}
		}
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
	app.SetRootWidget(hw)

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
