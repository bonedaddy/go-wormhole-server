package log

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

//Debug logs a debug message
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

//Debugf logs a debug message using formatting like fmt.Printf
func Debugf(str string, args ...interface{}) {
	logger.Debug(fmt.Sprintf(str, args...))
}

//Info logs an info message
func Info(args ...interface{}) {
	logger.Info(args...)
}

//Infof logs an info message using formatting like fmt.Printf
func Infof(str string, args ...interface{}) {
	logger.Info(fmt.Sprintf(str, args...))
}

//Warn logs a warning message
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

//Warnf logs a warning message using formatting lime fmt.Printf
func Warnf(str string, args ...interface{}) {
	logger.Warnf(fmt.Sprintf(str, args...))
}

//Error logs an error message
func Error(args ...interface{}) {
	logger.Error(args...)
}

//Errorf logs an error message using formatting like fmt.Printf
func Errorf(str string, args ...interface{}) {
	logger.Error(fmt.Sprintf(str, args...))
}

//Err logs an error message from an error object. Uses fmt.Printf
//for the bulk work, uses the last argument as the error object.
//Example:
//	err := someMethod()
//	log.Err("failed on id '%s'", 123, err)
func Err(msg string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"err": args[len(args)-1],
	}).Error(fmt.Sprintf(msg, args[:len(args)-1]...))
}
