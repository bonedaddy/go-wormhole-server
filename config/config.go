package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/chris-pikul/go-wormhole-server/log"
	"github.com/urfave/cli"
)

//RelayOptions holds the settings specific to the relay
//server operations
type RelayOptions struct {
	//Host portion for the servers to listen on.
	//Leaving this empty is fine as it will just use the default interface.
	Host string `json:"host"`

	//Port number for the server to listen on
	Port uint `json:"port"`

	//WelcomeMOTD set's the welcome message to be displayed on connecting
	//clients
	WelcomeMOTD string `json:"welcomeMOTD"`

	//WelcomeError is displayed to clients, and if provided will have
	//them disconnect immediately
	WelcomeError string `json:"welcomeError"`

	//DBFile path to the SQLite database file for the server to use
	DBFile string `json:"dbFile"`

	//AllowList allows clients to request a list of available nameplates
	AllowList bool `json:"allowList"`

	//CurrentVersion holds the current wormhole client version
	CurrentVersion string `json:"currentVersion"`

	//AdvertisedVersion holds the newest release version, which
	//will be advertised to clients to alert them of a new update
	AdvertisedVersion string `json:"advertisedVersion"`

	//CleaningInterval holds the time interval in which cleaning
	//operations should be ran
	CleaningInterval uint `json:"cleaningInterval"`

	//ChannelExpiration holds the time duration in which a channel
	//can exist without interaction before it is marked as dirty
	//and removed by cleaning. It is recommended this be larger
	//than the CleaningInterval field
	ChannelExpiration uint `json:"channelExpiration"` //TODO: This value is never used
}

//TransitOptions holds the settings specific to the transit
//(piping) server
type TransitOptions struct {
	//Host portion for the servers to listen on.
	//Leaving this empty is fine as it will just use the default interface.
	Host string `json:"host"`

	//Port number for the server to listen on
	Port uint `json:"port"`
}

const (
	//ModeBoth specifies to run both relay, and transit
	ModeBoth = "BOTH"

	//ModeRelay specifies to run only the relay portion
	ModeRelay = "RELAY"

	//ModeTransit specifies to run only the transit portion
	ModeTransit = "TRANSIT"
)

//Options is a JSON serializable object holding the configuration
//settings for running a Wormhole Server.
//
//These options can be loaded from file, or filled in from command line.
//The intended hierarchy is CLI options > File > Defaults
type Options struct {
	//Mode specifies in which mode should the server operate.
	//Options are:
	// - BOTH (default): Runs both the relay and transit servers on the
	//		same instance
	// - RELAY: Only run the relay server
	// - TRANSIT: Only run the transit server
	Mode string `json:"mode"`

	//Relay holds the relay portion options
	Relay RelayOptions `json:"relay"`

	//Transit holds the transit portion options
	Transit TransitOptions `json:"transit"`

	//Logging holds the options settings for logging operations
	Logging log.Options `json:"logging"`
}

//DefaultOptions contains the preset default options
//for a server.
var DefaultOptions = Options{
	Mode: ModeBoth,

	Relay: RelayOptions{
		Host:              "",
		Port:              4000,
		DBFile:            "./wormhole-relay.db",
		AllowList:         true,
		CleaningInterval:  5,
		ChannelExpiration: 11,
	},

	Transit: TransitOptions{
		Host: "",
		Port: 4001,
	},

	Logging: log.DefaultOptions,
}

var (
	//ErrOptionsMode validation error for mode
	ErrOptionsMode = errors.New("server mode invalid")

	//ErrOptionsCleaning validation error that cleaning interval
	//is larger then the channel expiration
	ErrOptionsCleaning = errors.New("cleaning interval should be less then channel expiration")
)

//Equals returns true if the supplied options matches these ones (this).
//Performs this as a deep-equals operation
func (o Options) Equals(opts Options) bool {
	return o.Mode == opts.Mode &&
		o.Relay == opts.Relay &&
		o.Transit == opts.Transit &&
		o.Logging.Equals(opts.Logging)
}

//Verify checks the Options fields for validity.
//Returns an error if a problem is incountered
func (o Options) Verify() error {
	if o.Mode != ModeBoth &&
		o.Mode != ModeRelay &&
		o.Mode != ModeTransit {
		return ErrOptionsMode
	}

	if o.Relay.CleaningInterval > o.Relay.ChannelExpiration {
		return ErrOptionsCleaning
	}

	return o.Logging.Verify()
}

//MergeFrom combines the fields from the supplied Options parameter
//into this object (smartly where applicable) and run Verify on itself,
//returning the validation error if any happened.
func (o *Options) MergeFrom(opt Options) error {
	o.Mode = opt.Mode

	o.Relay = opt.Relay
	o.Transit = opt.Transit

	err := o.Logging.MergeFrom(opt.Logging)
	if err != nil {
		return err
	}
	return o.Verify()
}

//ReadOptionsFromFile opens the provided JSON file and marshals the data
//into a Options object.
//Returns the results, and the first error encountered.
//The error is either validation error, or JSON encoding error.
func ReadOptionsFromFile(filename string) (Options, error) {
	res := DefaultOptions

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		return res, err
	}

	return res, res.Verify()
}

//NewOptions compiles the Options object from the provided sources.
//Will use a custom defaults, or if nil the DefaultOptions object is used.
//Then will search the fileName json file (if provided) for options.
//Then will combine the CLI options provided from main().
//These options cascade in order where applicable for the option.
//Will run the Options.Verify() method and return the error after compilation
func NewOptions(defaults *Options, filename string, ctx *cli.Context) (Options, error) {
	res := DefaultOptions
	if defaults != nil {
		res = *defaults
	}

	if len(filename) > 0 {
		fmt.Printf("reading configuration from '%s'\n", filename)
		file, err := ReadOptionsFromFile(filename)
		if err != nil {
			return res, err
		}
		err = res.MergeFrom(file)
		if err != nil {
			return res, err
		}
	}

	if ctx != nil {
		fmt.Printf("applying CLI options to configuration\n")
		applyCLIOptions(ctx, &res)
	}

	return res, res.Verify()
}

//applyCLIOptions writes the options presented in the CLI arguments to
//the provided ServerOptions object, overriding anything there previously
func applyCLIOptions(c *cli.Context, opts *Options) {
	if c == nil || opts == nil { //Safe-gaurd
		return
	}

	if c.String("config") != "" {
		//config file was used, ignore the flags
		return
	}

	opts.Relay.Host = c.String("relay-host")
	opts.Relay.Port = c.Uint("relay-port")
	opts.Transit.Host = c.String("transit-host")
	opts.Transit.Port = c.Uint("transit-port")

	opts.Relay.DBFile = c.String("db")

	if c.Bool("no-list") {
		opts.Relay.AllowList = false
	}

	if c.String("advert-version") != "" {
		opts.Relay.AdvertisedVersion = c.String("advert-version")
	}

	if c.Uint("cleaning") > 0 {
		ci := c.Uint("cleaning")
		opts.Relay.CleaningInterval = ci
	}

	if c.Uint("channel-exp") > 0 {
		ce := c.Uint("channel-exp")
		opts.Relay.ChannelExpiration = ce
	}

	opts.Logging.Path = c.String("log")

	if str := c.String("log-level"); str != "" {
		opts.Logging.Level = str
	}
	
	opts.Logging.BlurTimes = c.Uint("log-blur")
}
