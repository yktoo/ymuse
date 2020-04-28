package player

import (
	"github.com/op/go-logging"
	"github.com/yktoo/ymuse/internal/util"
)

// Package-wide Logger instance
var log *logging.Logger

func init() {
	// Init logging
	log = logging.MustGetLogger("player")
	logging.SetLevel(util.GetConfig().LogLevel, "player")
}

// errCheck() logs a warning if the error is not nil.
func errCheck(err error, message string) {
	if err != nil {
		log.Warning(message, err)
	}
}
