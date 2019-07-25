package log

import "errors"

const (
	//LevelDebug debug level logging, all messages outputted
	LevelDebug = "DEBUG"
	//LevelInfo info level logging, no debug information, a lot of info
	LevelInfo = "INFO"
	//LevelWarn warning level logging, only recovered errors, and fatal errors
	LevelWarn = "WARN"
	//LevelError error level logging, no other information other then fatal errors
	LevelError = "ERROR"
)

//Options holds the configuration settings
//for the logging operations. This is JSON serializable
//so we can load from a file.
type Options struct {
	//Path holds the file path to write logs too.
	//If this value is empty, then no file writing is
	//done and only STDOUT will be used
	Path string `json:"path"`

	//Level sets the logging level in which only
	//messages at, or above, this level will be witten.
	//The values expected are:
	//	DEBUG,INFO,WARN,ERROR
	//Where the default is INFO
	Level string `json:"level"`

	//Usage enables the logging of connection specific
	//messages
	Usage bool `json:"usage"`

	//BlurTimes tells the logging facilities to
	//round any access time logs to protect
	//user privacy
	BlurTimes uint `json:"blurTimes"`

	//ShowAddress enables the logging of connection
	//addresses during usage messages
	ShowAddress bool `json:"showRemoteAddresses"`
}

//DefaultOptions holds the default options
//for LoggingOptions objects
var DefaultOptions = Options{
	Path:        "",
	Level:       "DEBUG",
	Usage:       true,
	BlurTimes:   1,
	ShowAddress: true,
}

//ErrOptionLevel specifies the level field of the LoggingOptions object is invalid
var ErrOptionLevel = errors.New("invalid logging level option provided")

//Equals returns true if this object deep equals the provided one
func (o Options) Equals(opt Options) bool {
	return o.Path == opt.Path &&
		o.Level == opt.Level &&
		o.BlurTimes == opt.BlurTimes
}

//Verify confirms that all the options are valid
//within the set. If not returns an error declaring
//the problem.
func (o Options) Verify() error {
	if o.Level != LevelDebug &&
		o.Level != LevelInfo &&
		o.Level != LevelWarn &&
		o.Level != LevelError {
		return ErrOptionLevel
	}

	return nil
}

//MergeFrom combines the values from the supplied LoggingOptions
//parameter into this current options. Taking care to only override
//things needed. Will verify the results and return the object
//for any validation errors.
//
//Path will only be overriden if the supplied object has one
func (o *Options) MergeFrom(opt Options) error {
	if len(opt.Path) != 0 {
		o.Path = opt.Path
	}

	if opt.Level != "" {
		o.Level = opt.Level
	}

	o.BlurTimes = opt.BlurTimes

	return o.Verify()
}

//CombineOptions takes a variable amount of LoggingOptions objects
//and merges them into a single object, taking carefully merging them
//using LoggingOptions.MergeFrom() method. The starting object is DefaultLoggingOptions,
//so if no parameters are provided, the defaults are returned.
//Returns the new object, or an error if the final result does not
//pass validation. The returned error is the validation error as
//returned by LoggingOptions.Verify()
func CombineOptions(opts ...Options) (Options, error) {
	res := DefaultOptions

	var err error
	for _, opt := range opts {
		err = res.MergeFrom(opt)
		if err != nil {
			return res, err
		}
	}

	return res, nil
}
