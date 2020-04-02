package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/0xAX/notificator"
	"github.com/awnumar/memguard"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"github.com/pkg/profile"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/sprout-go"
	"git.sr.ht/~whereswaldon/sprout-go/watch"
	"git.sr.ht/~whereswaldon/wisteria/archive"
	"git.sr.ht/~whereswaldon/wisteria/widgets"
	wistTcell "git.sr.ht/~whereswaldon/wisteria/widgets/tcell"
)

var (
	version = "git"
	commit  = "unknown"
)

func CheckNotify() {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("notify-send"); err != nil {
			log.Println("WARNING: desktop notifications require `notify-send` to be installed")
		}
	}
}

func main() {
	// Safely terminate in case of an interrupt signal
	memguard.CatchInterrupt()

	// Purge the session when we return
	defer memguard.Purge()

	// need to find this value early in order to print it as the default value
	// for a flag.
	defaultConfig, err := DefaultConfigFilePath()
	if err != nil {
		log.Printf("Unable to determine default configuration file location: %v", err)
		return
	}
	defaultGrovePath, err := DefaultGrovePath()
	if err != nil {
		log.Printf("Unable to determine default grove location: %v", err)
		return
	}

	// declare flags
	configpath := flag.String("config", defaultConfig, "the configuration file to load")
	grovepath := flag.String("grove", defaultGrovePath, "path to the grove in use (directory of arbor history)")
	profiling := flag.Bool("profile", false, "enable CPU profiling (pprof file location will be logged)")
	insecure := flag.Bool("insecure", false, "disable TLS certificate validation when dialing relay addresses")
	nogpg := flag.Bool("nogpg", false, "disable the use of GPG for cryptography even when it is installed")
	testStartup := flag.Bool("test-startup", false, "run all the way through initializing the application, then exit gracefully. This flag is useful for automated testing of the startup configuration")
	printVersion := flag.Bool("version", false, "print version information and exit")
	flag.BoolVar(printVersion, "v", false, "print version information and exit")

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

	if *printVersion {
		fmt.Printf("%s version: %s commit: %s\n", os.Args[0], version, commit)
		return
	}
	// check whether we can send desktop notifications and warn if we can't
	CheckNotify()

	// ensure the grove path that we're working with actually exists
	if err := os.MkdirAll(*grovepath, 0770); err != nil {
		log.Fatalf("Failed creating the grove path: %v", err)
	}

	// make basic configuration
	config := NewConfig()
	config.GroveDirectory = *grovepath
	config.ConfigDirectory = filepath.Dir(*configpath)

	if *profiling {
		// profile to runtime directory chosen by config
		defer profile.Start(profile.ProfilePath(config.RuntimeDirectory)).Stop()
	}

	// create log widget early so we can provide it to log configuration
	logWidget := widgets.NewWriterWidget()
	logWidget.EnableCursor(true)

	// set up logging to runtime directory
	if err := config.StartLogging(logWidget); err != nil {
		log.Fatalf("Failed to configure logging: %v", err)
	}

	// use a grove rooted in our current working directory as our node storage
	groveStore, err := grove.New(*grovepath)
	if err != nil {
		log.Fatalf("Failed to create grove at %s: %v", *grovepath, err)
	}

	// Wrap store in CacheStore
	cache := forest.NewMemoryStore()
	store, err := forest.NewCacheStore(cache, groveStore)
	if err != nil {
		log.Fatal("Failed to wrap Store in CacheStore:", err)
	}

	runConfigurationWizard := false
	if err := config.LoadFromPath(*configpath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("Failed loading configuration file: %v", err)
		}
		runConfigurationWizard = true
	}

	// choose whether to enable gpg support.
	if *nogpg {
		// the flag to disable it should always win
		config.UseGPG = TristateFalse
	} else if config.UseGPG == TristateUndefined {
		// use gpg if it's available and the user hasn't opted out
		if GPGAvailable() {
			config.UseGPG = TristateTrue
		} else {
			config.UseGPG = TristateFalse
		}
	}

	wizard := &Wizard{
		Config:   config,
		Prompter: NewStdoutPrompter(os.Stdin, int(os.Stdin.Fd()), os.Stdout),
	}
	if runConfigurationWizard {
		// ask user for interactive configuration
		if err := wizard.Run(store); err != nil {
			log.Fatal("Error running configuration wizard:", err)
		}
		if err := config.Validate(); err != nil {
			log.Fatal("Error validating configuration:", err)
		}
		if err := config.SaveToPath(*configpath); err != nil {
			if !errors.Is(err, os.ErrExist) {
				log.Fatal("Error saving configuration:", err)
			}
			log.Printf("Choosing not to overwrite existing config file %s", *configpath)
		}
	}
	if config.UseGPG == TristateFalse && config.passphraseEnclave == nil {
		prompt := "Please enter your arbor identity passphrase (hit enter when finished):"
		if err := wizard.ConfigurePassphrase(prompt); err != nil {
			log.Fatalf("Failed to get arbor passphrase: %v", err)
		}
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
		tlsConfig := (*tls.Config)(nil)
		if *insecure {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		sprout.LaunchSupervisedWorker(done, address, subscriberStore, tlsConfig, log.New(log.Writer(), address+" ", log.Flags()))
	}

	// ensure its internal state is what we want
	history.Sort()

	// set up desktop notifications
	notify := notificator.New(notificator.Options{
		AppName: "Arbor",
	})

	// build an widget/application from existing views and services
	app := new(wistTcell.Application)
	hw, err := NewHistoryWidget(app, history, config, notify)
	if err != nil {
		log.Fatalf("Failed to create history widget: %v", err)
	}
	editorLayer := widgets.NewEphemeralEditor(hw)

	titlebar := views.NewSimpleStyledTextBar()
	titlebar.SetLeft("%Swisteria")
	titlebar.SetRight("%Sarrows or vi to move; enter to reply; c for new convo")
	titlebar.SetStyle(tcell.StyleDefault.Reverse(true))

	statusbar := widgets.NewStatusBar()
	// subscribe the status bar to events from the history widget
	hw.Watch(statusbar)
	hw.UpdateCursor() // set initial statusbar state

	switcher := widgets.NewSwitcher(app, editorLayer, logWidget)

	layout := views.NewBoxLayout(views.Vertical)
	layout.AddWidget(titlebar, 0)
	layout.AddWidget(switcher, 1)
	layout.AddWidget(statusbar, 0)
	app.SetRootWidget(layout)

	// watch the cwd for new nodes from other sources
	logger := log.New(log.Writer(), "", log.LstdFlags|log.Lshortfile)
	if _, err := watch.Watch(*grovepath, logger, hw.ReadMessageFile); err != nil {
		log.Fatal(err)
	}
	if *testStartup {
		go func() {
			time.Sleep(time.Second)
			app.Quit()
		}()
	}

	// configure the TUI screen for mouse support
	app.ConfigureScreen = func(screen tcell.Screen) {
		if screen.HasMouse() {
			log.Println("Enabling mouse support")
			screen.EnableMouse()
		} else {
			log.Println("Terminal does not advertise mouse support")
		}
	}

	// run the TUI
	if e := app.Run(); e != nil {
		log.Println(e.Error())
		os.Exit(1)
	}
}
