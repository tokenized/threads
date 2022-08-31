package threads

import (
	"context"
	"sync"
)

type Thread interface {
	// SetWait sets a wait group to use when executing the thread's function. It will add to it
	// before starting the function and call done when the function completes.
	SetWait(*sync.WaitGroup)

	// GetCompleteChannel returns a channel that will have the function's result written to it when
	// the thread's function completes.
	GetCompleteChannel() <-chan error

	// Start starts execution of the thread's function.
	Start(context.Context)

	// IsComplete returns true if the thread has completed executing.
	IsComplete() bool

	// Error returns any error returned from execution. Only call after a thread is complete.
	Error() error
}

type Threads []Thread

func (ts Threads) Start(ctx context.Context) {
	for _, thread := range ts {
		thread.Start(ctx)
	}
}

func (ts Threads) Errors() []error {
	var result []error
	for _, thread := range ts {
		if err := thread.Error(); err != nil {
			result = append(result, err)
		}
	}

	return result
}
