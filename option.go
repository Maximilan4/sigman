package sigman

import (
	"log"
	"os"
)

type Option func(*Sigman)

func Signals(signals ...os.Signal) Option {
	return func(m *Sigman) {
		m.signals = signals
	}
}

func Logger(logger *log.Logger) Option {
	return func(m *Sigman) {
		m.logger = logger
	}
}
