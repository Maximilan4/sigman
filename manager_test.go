package sigman

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func Test_getFName(t *testing.T) {
	f := func() {}

	name := getFName(f)
	if name != "github.com/Maximilan4/sigman.Test_getFName.func1" {
		t.Error("Expected github.com/Maximilan4/sigman.Test_getFName.func1, got ", name)
	}
}

func TestNewManager(t *testing.T) {

	t.Run("FullOpts", func(t *testing.T) {
		logger := log.New(os.Stdout, "sigman", log.LstdFlags)
		man := New(Logger(logger), Signals(syscall.SIGUSR1, syscall.SIGUSR2))

		if logger != man.logger {
			t.Error("Logger is no the same")
		}

		if len(man.signals) != 2 {
			t.Error("signals was modified")
		}
	})

	t.Run("WithoutOpts", func(t *testing.T) {
		man := New()

		if log.Default() != man.logger {
			t.Error("Logger is no the same")
		}

		if len(man.signals) != len(DefaultSignals) {
			t.Error("signals was modified")
		}
	})
}

func TestManager_Add(t *testing.T) {
	f := func(ctx context.Context, sig os.Signal, m *Sigman) error {
		return nil
	}
	sigs := []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2}
	logger := log.New(io.Discard, "sigman", log.LstdFlags)
	man := New(Logger(logger), Signals(sigs...))

	t.Run("Err Nil f", func(t *testing.T) {
		err := man.Add(nil)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("Err empty sigs", func(t *testing.T) {
		err := man.Add(f)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("Correct", func(t *testing.T) {
		err := man.Add(f, sigs...)
		if err != nil {
			t.Error(err)
		}
		for _, sig := range sigs {
			if len(man.handlers[sig]) != 1 {
				t.Error("sig", sig, "handler is not correct registered")
			}

			if reflect.DeepEqual(man.handlers[sig][0], f) {
				t.Error("sig", sig, "is not registered")
			}
		}

	})

}

func TestManager_RemoveAll(t *testing.T) {
	f := func(ctx context.Context, sig os.Signal, m *Sigman) error {
		return nil
	}
	sigs := []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2}
	logger := log.New(io.Discard, "sigman", log.LstdFlags)
	man := New(Logger(logger), Signals(sigs...))
	_ = man.Add(f, sigs...)
	man.RemoveAll()
	if len(man.handlers) != 0 {
		t.Error("handlers was not removed")
	}
}

func TestManager_Remove(t *testing.T) {
	f := func(ctx context.Context, sig os.Signal, m *Sigman) error {
		return nil
	}
	sigs := []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2}
	logger := log.New(io.Discard, "sigman", log.LstdFlags)
	man := New(Logger(logger), Signals(sigs...))
	_ = man.Add(f, sigs...)

	t.Run("Empty sigs", func(t *testing.T) {
		man.Remove()
		for _, sig := range sigs {
			if _, ok := man.handlers[sig]; !ok {
				t.Error("signal was removed", sig)
			}
		}
	})

	t.Run("remove USR1", func(t *testing.T) {
		man.Remove(syscall.SIGUSR1)
		if _, ok := man.handlers[syscall.SIGUSR1]; ok {
			t.Error("SIGUSR1 has not been removed")
		}
	})

}

func TestManager_WaitStop(t *testing.T) {
	var called int
	f := func(ctx context.Context, sig os.Signal, m *Sigman) error {
		called++
		return nil
	}
	sigs := []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2}
	logger := log.New(io.Discard, "sigman", log.LstdFlags)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	man := New(Logger(logger), Signals(syscall.SIGIO, syscall.SIGUSR1, syscall.SIGUSR2))
	_ = man.Add(f, sigs...)

	time.AfterFunc(time.Second/2, func() {
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGUSR1); err != nil {
			t.Error(err)
		}

		if err := syscall.Kill(syscall.Getpid(), syscall.SIGUSR2); err != nil {
			t.Error(err)
		}
		time.Sleep(100 * time.Millisecond)
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGIO); err != nil {
			t.Error(err)
		}
	})

	time.AfterFunc(2*time.Second, func() {
		if err := man.Stop(); err != nil {
			t.Error(err)
		}
	})

	if err := man.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}

	if called != 2 {
		t.Error("handler was not called 2 times")
	}
}

func TestManager_StopOnSignal(t *testing.T) {
	logger := log.New(io.Discard, "sigman", log.LstdFlags)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	man := New(Logger(logger), Signals(syscall.SIGTERM))
	_ = man.Add(Shutdown, syscall.SIGTERM)

	time.AfterFunc(time.Second/2, func() {
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
			t.Error(err)
		}
	})

	if err := man.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
}

func TestManager_WaitErr(t *testing.T) {
	logger := log.New(io.Discard, "sigman", log.LstdFlags)
	man := New(Logger(logger), Signals(syscall.SIGTERM))

	t.Run("BadCtx", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		expErr := errors.New("early cancel")
		cancel(expErr)
		if err := man.Wait(ctx); !errors.Is(err, expErr) {
			t.Error(err)
		}
	})

	t.Run("SecondWaitCall", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(100*time.Millisecond, func() {
			if err := man.Wait(ctx); !strings.Contains(err.Error(), "already started") {
				t.Error(err)
			}
			cancel()
		})

		if err := man.Wait(ctx); !errors.Is(err, context.Canceled) {
			t.Error(err)
		}

	})
	// time.AfterFunc(time.Second/2, func() {
	// 	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
	// 		t.Error(err)
	// 	}
	// })

}
