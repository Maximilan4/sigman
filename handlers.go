package sigman

import (
	"context"
	"os"
)

func defaultHandler(_ context.Context, sig os.Signal, m *Sigman) error {
	m.logger.Printf("incoming signal %s, no handler found, skip", sig)
	return nil
}

func Shutdown(_ context.Context, _ os.Signal, m *Sigman) error {
	return m.Stop()
}
