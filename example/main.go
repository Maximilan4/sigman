package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/Maximilan4/sigman"
)

// output:
// sigman	2025/03/28 00:47:28.062965 manager.go:122: assigned handler 'main.main.func1' to sig 'user defined signal 1'
// sigman	2025/03/28 00:47:28.063303 manager.go:122: assigned handler 'main.main.func1' to sig 'user defined signal 2'
// sigman	2025/03/28 00:47:28.166688 manager.go:183: got 'user defined signal 2' signal, executing 1 handlers
// sigman	2025/03/28 00:47:28.166721 manager.go:187: exec main.main.func1: err=signal user defined signal 2 not supported
//
// sigman	2025/03/28 00:47:28.266411 manager.go:183: got 'user defined signal 1' signal, executing 1 handlers
// sigman	2025/03/28 00:47:28.266465 main.go:35: received user defined signal 1
// sigman	2025/03/28 00:47:28.266478 manager.go:187: exec main.main.func1: err=<nil>
// 2025/03/28 00:47:29 context deadline exceeded
func main() {
	logger := log.New(os.Stdout, "sigman ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	sm := sigman.New(
		sigman.Logger(logger),                    // not required, log.Default() will be used instead
		sigman.Signals(sigman.DefaultSignals...), // not required, this is a default behaviour
	)

	// add a handler func for usr signals
	err := sm.Add(func(ctx context.Context, sig os.Signal, m *sigman.Sigman) error {
		if sig != syscall.SIGUSR1 {
			return fmt.Errorf("signal %v not supported\n", sig)
		}
		logger.Println("received", sig)
		return nil
	}, syscall.SIGUSR1, syscall.SIGUSR2)
	if err != nil {
		log.Fatal(err)
	}

	// this adds default stop handler - it calls sm.Stop() for stopping manager
	_ = sm.Add(sigman.Shutdown, syscall.SIGTERM, syscall.SIGINT)

	// emulate signal call
	time.AfterFunc(100*time.Millisecond, func() {
		syscall.Kill(syscall.Getpid(), syscall.SIGUSR2)
	})
	time.AfterFunc(200*time.Millisecond, func() {
		syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	})

	// ctx with timeout just for example
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err = sm.Wait(ctx); err != nil {
		log.Fatal(err)
	}
}
