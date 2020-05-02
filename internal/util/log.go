package util

import (
	"errors"
	"fmt"
	"github.com/op/go-logging"
)

// Package-wide Logger instance
var log *logging.Logger

func init() {
	// Init logging
	log = logging.MustGetLogger("util")
	logging.SetLevel(GetConfig().LogLevel, "util")
}

// errCheck() logs a warning if the error is not nil.
func errCheck(err error, message string) bool {
	if err != nil {
		log.Warning(errors.New(fmt.Sprintf("%v: %v", message, err)))
		return true
	}
	return false
}
