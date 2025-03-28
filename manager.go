package sigman

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
)

type (
	// managerCtxKey - type for storing manager in context
	managerCtxKey struct{}

	// SignalHandlerFunc - basic func alias for signal`s handler register
	SignalHandlerFunc func(ctx context.Context, sig os.Signal, m *Sigman) error

	// Sigman - signal manager basic structure for storing signal handlers in one place
	Sigman struct {
		handlers  map[os.Signal][]SignalHandlerFunc
		signals   []os.Signal
		ch        chan os.Signal
		ctx       context.Context
		ctxCancel context.CancelFunc
		logger    *log.Logger
		mut       *sync.Mutex
		started   bool
	}
)

var (
	CtxKey = managerCtxKey{}
)

// getFName - helper func for getting name of given handler func
func getFName(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// New - creates a new instance on signal manager with given options
func New(options ...Option) *Sigman {
	m := Sigman{}
	for _, option := range options {
		option(&m)
	}

	if m.signals == nil {
		m.signals = DefaultSignals
	}

	if m.logger == nil {
		m.logger = log.Default()
	}

	m.ch = make(chan os.Signal, 1)
	signal.Notify(m.ch, m.signals...)

	m.handlers = make(map[os.Signal][]SignalHandlerFunc, len(m.signals))
	m.mut = new(sync.Mutex)
	return &m
}

// Add - save given func as a handler for one or more signals
func (sm *Sigman) Add(f SignalHandlerFunc, signals ...os.Signal) error {
	if f == nil {
		return errors.New("nil handler given")
	}

	if len(signals) == 0 {
		return errors.New("min 1 signal for handler required")
	}

	sm.mut.Lock()
	defer sm.mut.Unlock()

	fName := getFName(f)
	for _, sig := range signals {
		if _, ok := sm.handlers[sig]; !ok {
			sm.handlers[sig] = make([]SignalHandlerFunc, 0, 1)
		}
		sm.handlers[sig] = append(sm.handlers[sig], f)
		sm.logger.Printf("assigned handler '%s' to sig '%s'\n", fName, sig)
	}

	return nil
}

// RemoveAll - removes all registered signal handlers
func (sm *Sigman) RemoveAll() {
	sm.mut.Lock()
	defer sm.mut.Unlock()

	sm.handlers = make(map[os.Signal][]SignalHandlerFunc, len(sm.signals))
	sm.logger.Printf("removed handlers for all signals\n")
	return
}

// Remove - remove all registered signal handlers for given signals
func (sm *Sigman) Remove(signals ...os.Signal) {
	if len(signals) == 0 {
		return
	}

	sm.mut.Lock()
	defer sm.mut.Unlock()

	for _, sig := range signals {
		delete(sm.handlers, sig)
		sm.logger.Printf("removed handlers of sig '%s'\n", sig)
	}
}

// Wait - starts signal handling in blocking mode
func (sm *Sigman) Wait(ctx context.Context) error {
	// ctx with err is possibly closed, return
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("bad ctx given: %w", context.Cause(ctx))
	}

	sm.mut.Lock()
	if sm.started {
		return errors.New("already started")
	}
	sm.started = true

	// save wrapped ctx as a current state
	sm.ctx, sm.ctxCancel = context.WithCancel(ctx)
	sm.ctx = context.WithValue(sm.ctx, CtxKey, &sm)
	sm.mut.Unlock()

	for {
		select {
		case <-sm.ctx.Done():
			return context.Cause(sm.ctx)
		case sig := <-sm.ch:
			sm.mut.Lock()
			if _, ok := sm.handlers[sig]; !ok {
				_ = defaultHandler(sm.ctx, sig, sm) // no err for default handler
				sm.mut.Unlock()
				continue
			}

			sm.logger.Printf("got '%s' signal, executing %d handlers\n", sig, len(sm.handlers[sig]))
			var err error
			for _, handler := range sm.handlers[sig] {
				err = handler(sm.ctx, sig, sm)
				sm.logger.Printf("exec %s: err=%v\n", getFName(handler), err)
			}
			sm.mut.Unlock()
		}
	}
}

// Start - run wait process in background
func (sm *Sigman) Start(ctx context.Context) {
	go func() {
		if err := sm.Wait(ctx); err != nil {
			sm.logger.Printf("wait err: %v\n", err)
		}
	}()
}

// Stop - stops current process and cancels internal context
func (sm *Sigman) Stop() error {
	if sm.ctx == nil {
		return nil
	}

	sm.ctxCancel()
	err := context.Cause(sm.ctx)
	if errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}

func (sm *Sigman) Close() error {
	sm.mut.Lock()
	defer sm.mut.Unlock()
	signal.Stop(sm.ch)
	close(sm.ch)
	return sm.Stop()
}

// Ctx - returns a manager context. Can be nil, if manager was not started
func (sm *Sigman) Ctx() context.Context {
	return sm.ctx
}
