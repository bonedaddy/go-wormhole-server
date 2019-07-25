package relay

import (
	"time"

	"github.com/chris-pikul/go-wormhole-server/config"
	"github.com/chris-pikul/go-wormhole-server/log"
	"github.com/sirupsen/logrus"
)

func prepLog(c *Client) *logrus.Entry {
	var l = log.Get().WithField("usage", "relay")
	if config.Opts.Logging.BlurTimes {
		l = l.WithTime(time.Now().Truncate(time.Second))
	}
	if config.Opts.Logging.ShowAddress {
		l = l.WithField("remote-addr", c.conn.RemoteAddr())
	}
	return l
}

//LogDebug is a convenience wrapper for logging
//usage statistics given the relay server settings
func LogDebug(c *Client, args ...interface{}) {
	if config.Opts == nil || !config.Opts.Logging.Usage {
		return
	}

	prepLog(c).Debug(args...)
}

//LogDebugf is a convenience wrapper for logging
//usage statistics given the relay server settings
func LogDebugf(c *Client, fmt string, args ...interface{}) {
	if config.Opts == nil || !config.Opts.Logging.Usage {
		return
	}

	prepLog(c).Debugf(fmt, args...)
}

//LogInfo is a convenience wrapper for logging
//usage statistics given the relay server settings
func LogInfo(c *Client, args ...interface{}) {
	if config.Opts == nil || !config.Opts.Logging.Usage {
		return
	}

	prepLog(c).Info(args...)
}

//LogInfof is a convenience wrapper for logging
//usage statistics given the relay server settings
func LogInfof(c *Client, fmt string, args ...interface{}) {
	if config.Opts == nil || !config.Opts.Logging.BlurTimes {
		return
	}

	prepLog(c).Infof(fmt, args...)
}

//LogWarn is a convenience wrapper for logging warnings
//with usage statistics
func LogWarn(c *Client, fmt string, args ...interface{}) {
	if config.Opts == nil || !config.Opts.Logging.BlurTimes {
		return
	}

	prepLog(c).Warn(args...)
}

//LogWarnf is a convenience wrapper for logging warnings
//with usage statistics
func LogWarnf(c *Client, fmt string, args ...interface{}) {
	if config.Opts == nil || !config.Opts.Logging.BlurTimes {
		return
	}

	prepLog(c).Warnf(fmt, args...)
}

//LogError is a convenience wrapper for logging errors
//with usage statistics
func LogError(c *Client, fmt string, args ...interface{}) {
	if config.Opts == nil || !config.Opts.Logging.BlurTimes {
		return
	}

	prepLog(c).Error(args...)
}

//LogErrorf is a convenience wrapper for logging errors
//with usage statistics
func LogErrorf(c *Client, fmt string, args ...interface{}) {
	if config.Opts == nil || !config.Opts.Logging.BlurTimes {
		return
	}

	prepLog(c).Errorf(fmt, args...)
}

//LogErr is a convenience wrapper for logging errors
//with usage statistics
func LogErr(c *Client, msg string, err error) {
	if config.Opts == nil || !config.Opts.Logging.BlurTimes {
		return
	}

	prepLog(c).WithError(err).Error(msg)
}
