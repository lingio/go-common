package common

import (
	"context"
)

type (
	Sidecar struct {
		ctx    context.Context
		cancel context.CancelFunc
		done   chan error
	}
)

// NewSidecar immediately starts `fn` in a new goroutine, passing
// a Sidecar-owned context so a subsequent call to `Shutdown` can
// cancel it.
//
// Example:
//
//	s := NewSidecar(func(ctx context.Context) error {
//	    for {
//			select {
//			case <-time.After(time.Minute*5):
//				if err := doSomethingCancellable(ctx); err != nil {
//					log.Println("unrecoverable error:", err)
//					return err
//				}
//			case <-ctx.Done():
//				return nil
//		}
//	})
//	...
//	s.Shutdown(ctxWithTimeout)
func NewSidecar(fn func(context.Context) error) *Sidecar {
	done := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer close(done)
		done <- fn(ctx)
	}()

	return &Sidecar{
		ctx:    ctx,
		cancel: cancel,
		done:   done,
	}
}

// Shutdown cancels the internal context which was passed to the wrapped `fn`
// and blocks until either `ctx` is cancelled or the sidecar has exited.
func (ss *Sidecar) Shutdown(ctx context.Context) (err error) {
	ss.cancel()

	select {
	case err = <-ss.done:
		return
	case <-ctx.Done():
		return ctx.Err()
	}
}
