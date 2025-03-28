package sigman

import (
	"context"
	"os"
)

var std = New()

// Default - returns a global instance of manager
func Default() *Sigman {
	return std
}

// Add - add handler func to std manager
func Add(f SignalHandlerFunc, signals ...os.Signal) error {
	return std.Add(f, signals...)
}

// RemoveAll - remove all handlers from std manager
func RemoveAll() {
	std.RemoveAll()
}

// Remove - remove handlers for specific signals from std manager
func Remove(signals ...os.Signal) {
	std.Remove(signals...)
}

// Start - start waiting for signals
func Start(ctx context.Context) {
	std.Start(ctx)
}

// Stop - stop signal waiting
func Stop() error {
	return std.Stop()
}

// Ctx - returns a ctx from signal manager, if it was started
func Ctx() context.Context {
	if !std.started {
		return nil
	}

	return std.Ctx()
}
