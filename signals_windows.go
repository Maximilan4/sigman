package sigman

import (
	"os"

	"golang.org/x/sys/windows"
)

var DefaultSignals = []os.Signal{
	windows.SIGHUP,
	windows.SIGINT,
	windows.SIGQUIT,
	windows.SIGILL,
	windows.SIGTRAP,
	windows.SIGABRT,
	windows.SIGBUS,
	windows.SIGFPE,
	windows.SIGKILL,
	windows.SIGSEGV,
	windows.SIGPIPE,
	windows.SIGALRM,
	windows.SIGTERM,
}
