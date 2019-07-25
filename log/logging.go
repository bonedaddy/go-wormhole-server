package log

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()
var logBlur = DefaultOptions.BlurTimes

//Initialize sets up the logging interface for use without the server
func Initialize(cfg Options) error {
	//Double check the config is valid
	if err := cfg.Verify(); err != nil {
		return err
	}

	//Switch on the level
	switch cfg.Level {
	case LevelDebug:
		logger.Level = logrus.DebugLevel
	case LevelInfo:
		logger.Level = logrus.InfoLevel
	case LevelWarn:
		logger.Level = logrus.WarnLevel
	case LevelError:
		logger.Level = logrus.ErrorLevel
	default:
		logger.Level = logrus.InfoLevel
	}

	//Use a file if we need too
	if cfg.Path != "" {
		//TODO: Support more logging targets like streams, urls, etc?
		f, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0750)
		if err != nil {
			return fmt.Errorf("failed to open log file for writing\nerror: %s", err.Error())
		}

		logger.Out = f
	}

	//Set the blur format
	logBlur = cfg.BlurTimes

	return nil
}

//Get returns the underlying logrus logger object
func Get() *logrus.Logger {
	return logger
}
