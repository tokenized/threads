package threads

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/tokenized/logger"

	"github.com/pkg/errors"
)

// UninterruptableFunction is a function that can be stopped by external means like closing a
// channel or connection.
type UninterruptableFunction func(ctx context.Context) error

// UnterruptableThread thread is a thread that is stopped by external means like closing a channel
// or connection.
type UnterruptableThread struct {
	name     string
	function UninterruptableFunction

	complete   chan error
	wait       *sync.WaitGroup
	err        error
	isComplete bool
	lock       sync.Mutex
}

func NewUninterruptableThread(name string, function UninterruptableFunction) *UnterruptableThread {
	return &UnterruptableThread{
		name:     name,
		function: function,
	}
}

func NewUninterruptableThreadComplete(name string, function UninterruptableFunction,
	wait *sync.WaitGroup) (*UnterruptableThread, <-chan error) {

	complete := make(chan error, 1)
	return &UnterruptableThread{
		name:     name,
		function: function,
		complete: complete,
		wait:     wait,
	}, complete
}

func (t *UnterruptableThread) SetWait(wait *sync.WaitGroup) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.wait = wait
}

func (t *UnterruptableThread) GetCompleteChannel() <-chan error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// use buffered with size of one so it doesn't wait on write if there is no reader waiting
	complete := make(chan error, 1)
	t.complete = complete
	return complete
}

func (t *UnterruptableThread) Start(ctx context.Context) {
	t.lock.Lock()
	name := t.name
	function := t.function
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

			if complete != nil {
				close(complete)
			}

			if wait != nil {
				wait.Done()
			}
		}()

		err := function(ctx)

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

func (t *UnterruptableThread) IsComplete() bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.isComplete
}

func (t *UnterruptableThread) Error() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.err
}
