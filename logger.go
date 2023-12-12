package main

import (
	"os"
	"strconv"

	"github.com/phuslu/log"
)

func initLogger() {
	if log.IsTerminal(os.Stderr.Fd()) {
		log.DefaultLogger = log.Logger{
			Level:      getLogLevel(),
			TimeFormat: "15:04:05",
			Caller:     1,
			Writer: &log.ConsoleWriter{
				ColorOutput:    true,
				QuoteString:    true,
				EndWithMessage: true,
			},
		}
	} else {
		log.DefaultLogger = log.Logger{
			Level:      getLogLevel(),
			Caller:     1,
			TimeField:  "date",
			TimeFormat: "2006-01-02T15:04:05.999Z07:00",
			Writer:     &log.IOWriter{os.Stdout},
		}
	}
}

func getLogLevel() log.Level {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		return log.InfoLevel
	}
	intLevel, err := strconv.Atoi(logLevel)
	if err != nil {
		return log.InfoLevel
	}
	return log.Level(intLevel)
}
