package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chris-pikul/go-wormhole-server/config"
	"github.com/chris-pikul/go-wormhole-server/log"
	"github.com/chris-pikul/go-wormhole-server/relay"

	"github.com/urfave/cli"
)

const (
	//Version holds the CLI application version
	Version = "0.1.0"
)

const usageText = `wormhole-server [command] [global options...]

   Default command is "serve".
   If the config option is provided, then all the other options are
   ignored and the json file is used instead.
   NOTE: since the config can provide the server mode, the different
   mode commands are ignored
`

var (
	cfg config.Options

	chanQuit = make(chan bool)
)

func main() {
	app := cli.NewApp()
	app.Name = "Magic Wormhole Server"
	app.Usage = "facilitate the usage of wormhole protocol for client file/text transfering"
	app.UsageText = usageText
	app.HelpName = "wormhole-server"
	app.Version = Version
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Chris Pikul",
			Email: "chris-pikul@gmail.com",
		},
	}

	//NOTE: Major, no real way to tell if these are CLI defaults,
	//or DefaultServerOptions defaults, so because of the build order
	//of options the CLI just dictates the final object irregardless
	//of if a configuration file is used.
	//For this reason, if a config file is provided, the options are ignored
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "configuration JSON file to use instead of options (empty = no config)",
		},

		cli.StringFlag{
			Name:  "relay-host",
			Usage: "host address or IP for the listening interface",
		},
		cli.UintFlag{
			Name:  "relay-port",
			Usage: "port number to listen on",
			Value: 4000,
		},

		cli.StringFlag{
			Name:  "transit-host",
			Usage: "host address or IP for the listening interface",
		},
		cli.UintFlag{
			Name:  "transit-port",
			Usage: "port number to listen on",
			Value: 4001,
		},

		cli.StringFlag{
			Name:  "db, d",
			Usage: "path to SQLite database file",
			Value: "wormhole-relay.db",
		},
		cli.BoolFlag{
			Name:  "no-list",
			Usage: "disable the 'list' request",
		},
		cli.StringFlag{
			Name:  "advert-version",
			Usage: "version to recommend to clients",
		},

		cli.UintFlag{
			Name:  "cleaning, C",
			Usage: "time interval inbetween cleaning channels in minutes",
			Value: 5,
		},
		cli.UintFlag{
			Name:  "channel-exp, e",
			Usage: "channel expiration time in minutes (should be larger then cleaning period)",
			Value: 11,
		},

		cli.StringFlag{
			Name:  "log, l",
			Usage: "file to write usage/error logs to (empty does not write logs)",
		},
		cli.StringFlag{
			Name:  "log-level, L",
			Usage: "logging level to use options are [DEBUG|INFO|WARN|ERROR]",
			Value: "INFO",
		},
		cli.UintFlag{
			Name:  "log-blur",
			Usage: "round out access times to seconds provided in logging to improve privacy",
			Value: 1,
		},
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name:   "serve",
			Usage:  "serve both relay, and transit requests (default command)",
			Action: runServer,
		},

		cli.Command{
			Name:   "clean",
			Usage:  "clears the SQLite database file",
			Action: runClean,
		},

		cli.Command{
			Name:   "relay",
			Usage:  "run as relay server (rendezvous) only",
			Action: runRelay,
		},

		cli.Command{
			Name:   "transit",
			Usage:  "run as transit server (piping) only",
			Action: runTransit,
		},
	}

	app.Action = runServer

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

//common initialization procedures
func initialize(c *cli.Context) error {
	var err error

	//Load the configuration (from file if needed)
	cfgFile := c.String("config")
	cfg, err = config.NewOptions(nil, cfgFile, c)
	if err != nil {
		return fmt.Errorf("failed to parse configuration options; error = %s", err.Error())
	}
	config.Opts = &cfg //Make it global

	//Startup logging as soon as possible
	if err := log.Initialize(cfg.Logging); err != nil {
		return fmt.Errorf("failed to startup server due to logging issue; error = %s", err.Error())
	}
	log.Info("initialized logging")

	return nil
}

//performs the shutdown steps for graceful closing of the servers
func shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	relay.Shutdown(ctx)
}

//holds the main thread until either an interrupt from OS, or the chanQuit receives a message
func blockUntilSignalOrTermination() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	//Block until terminated
	select {
	case <-sigChan:
		log.Info("closing due to interrupt")
	case <-chanQuit:
		log.Info("closing from quit message")
	}
}

func runServer(c *cli.Context) error {
	err := initialize(c)
	if err != nil {
		return err
	}

	err = relay.Initialize()
	if err != nil {
		return err
	}
	relay.Start()

	blockUntilSignalOrTermination()
	shutdown()

	return nil
}

func runClean(c *cli.Context) error {
	fmt.Println("cleaning database")
	return nil
}

func runRelay(c *cli.Context) error {
	fmt.Println("running wormhole relay (only) server")
	return nil
}

func runTransit(c *cli.Context) error {
	fmt.Println("running wormhole transit (only) server")
	return nil
}
