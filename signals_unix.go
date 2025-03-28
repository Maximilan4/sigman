//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package sigman

import (
	"os"
	"slices"
	"syscall"

	"golang.org/x/sys/unix"
)

var DefaultSignals = loadDefaultSignals()

func loadDefaultSignals() []os.Signal {
	slice := make([]os.Signal, 0)
	for i := syscall.Signal(0); i < syscall.Signal(255); i++ {
		name := unix.SignalName(i)
		if name != "" {
			continue
		}

		slice = append(slice, i)
	}

	return slices.Clip(slice)
}
