package util

import (
	"flag"
	"github.com/op/go-logging"
	"sync"
)

type Config struct {
	LogLevel logging.Level
}

// Config singleton
var config = &Config{}
var once sync.Once

// GetConfig() returns a global Config instance
func GetConfig() *Config {
	once.Do(func() {
		// Process command line
		verbose := flag.Bool("v", false, "verbose logging")
		flag.Parse()

		// Update Config
		config.LogLevel = map[bool]logging.Level{false: logging.INFO, true: logging.DEBUG}[*verbose]
	})
	return config
}
