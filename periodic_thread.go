package threads

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/tokenized/logger"

	"github.com/pkg/errors"
)

// PeriodicThread is a thread that calls a function periodically.
type PeriodicThread struct {
	name     string
	function UninterruptableFunction
	period   time.Duration

	interrupt  chan interface{}
	complete   chan error
	wait       *sync.WaitGroup
	err        error
	isComplete bool
	lock       sync.Mutex
}

func NewPeriodicThread(name string, function UninterruptableFunction,
	period time.Duration) *PeriodicThread {

	return &PeriodicThread{
		name:     name,
		function: function,
		period:   period,
	}
}

func NewPeriodicThreadComplete(name string, function UninterruptableFunction, period time.Duration,
	wait *sync.WaitGroup) (*PeriodicThread, <-chan error) {

	complete := make(chan error, 1)
	return &PeriodicThread{
		name:     name,
		function: function,
		period:   period,
		complete: complete,
		wait:     wait,
	}, complete
}

func (t *PeriodicThread) SetWait(wait *sync.WaitGroup) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.wait = wait
}

func (t *PeriodicThread) GetCompleteChannel() <-chan error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// use buffered with size of one so it doesn't wait on write if there is no reader waiting
	complete := make(chan error, 1)
	t.complete = complete
	return complete
}

func (t *PeriodicThread) Start(ctx context.Context) {
	interrupt := make(chan interface{}, 1)

	t.lock.Lock()
	name := t.name
	function := t.function
	period := t.period
	t.interrupt = interrupt
	complete := t.complete
	wait := t.wait
	t.lock.Unlock()

	caller := logger.GetCaller(1) // use caller of thread start rather than thread file

	if wait != nil {
		wait.Add(1)
	}
	go func() {
		logger.LogDepthWithFields(ctx, logger.LevelDebug, caller, nil, "Starting: %s (thread)",
			name)

		defer func() {
			// Check for a panic in this thread.
			if pnc := recover(); pnc != nil {
				logger.LogDepthWithFields(ctx, logger.LevelError, caller, []logger.Field{
					logger.String("stack", string(debug.Stack())),
				}, "%s (thread): Panic: %s", name, pnc)

				err := fmt.Errorf("%s (thread): panic: %s", name, pnc)

				t.lock.Lock()
				t.err = err
				t.lock.Unlock()

				if complete != nil {
					complete <- err
				}
			}

			// Mark thread complete
			if complete != nil {
				close(complete)
			}

			// Mark wait as done
			if wait != nil {
				wait.Done()
			}
		}()

		var err error
		stop := false
		for !stop {
			err = function(ctx)
			if err != nil {
				stop = true
			} else {
				select {
				case <-interrupt:
					stop = true
				case <-time.After(period):
				}
			}
		}

		t.lock.Lock()
		t.err = err
		t.isComplete = true
		t.lock.Unlock()

		if complete != nil {
			complete <- err
		}

		if err == nil {
			logger.LogDepthWithFields(ctx, logger.LevelDebug, caller, nil,
				"Finished: %s (thread)", name)
		} else if errors.Cause(err) == Interrupted {
			logger.LogDepthWithFields(ctx, logger.LevelDebug, caller, nil,
				"Finished: %s (thread) : %s", name, err)
		} else {
			logger.LogDepthWithFields(ctx, logger.LevelVerbose, caller, nil,
				"Finished: %s (thread): %s", name, err)
		}
	}()
}

func (t *PeriodicThread) Stop(ctx context.Context) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.interrupt != nil {
		close(t.interrupt)
		t.interrupt = nil
	}
}

func (t *PeriodicThread) IsComplete() bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.isComplete
}

func (t *PeriodicThread) Error() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.err
}
