package threads

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/tokenized/logger"

	"github.com/pkg/errors"
)

// InterruptableFunction is a function that returns when an "interrupt" channel is closed. It should
// select on interrupt and end the function when it is closed, if not before.
type InterruptableFunction func(ctx context.Context, interrupt <-chan interface{}) error

// InterruptableThread is a thread that is stopped by closing an "interrupt" channel.
type InterruptableThread struct {
	name     string
	function InterruptableFunction

	interrupt  chan interface{}
	complete   chan error
	wait       *sync.WaitGroup
	err        error
	isComplete bool
	lock       sync.Mutex
}

func NewInterruptableThread(name string, function InterruptableFunction) *InterruptableThread {
	return &InterruptableThread{
		name:     name,
		function: function,
	}
}

func NewInterruptableThreadComplete(name string, function InterruptableFunction,
	wait *sync.WaitGroup) (*InterruptableThread, <-chan error) {

	complete := make(chan error, 1)
	return &InterruptableThread{
		name:     name,
		function: function,
		complete: complete,
		wait:     wait,
	}, complete
}

func (t *InterruptableThread) SetWait(wait *sync.WaitGroup) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.wait = wait
}

func (t *InterruptableThread) GetCompleteChannel() <-chan error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// use buffered with size of one so it doesn't wait on write if there is no reader waiting
	complete := make(chan error, 1)
	t.complete = complete
	return complete
}

func (t *InterruptableThread) Start(ctx context.Context) {
	interrupt := make(chan interface{}, 1)

	t.lock.Lock()
	name := t.name
	t.interrupt = interrupt
	function := t.function
	complete := t.complete
	wait := t.wait
	t.lock.Unlock()

	caller := logger.GetCaller(1) // use caller of thread start rather than thread file

	if wait != nil {
		wait.Add(1)
	}
	go func() {
		logger.LogDepthWithFields(ctx, logger.LevelVerbose, caller, nil, "Starting: %s (thread)",
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

		err := function(ctx, interrupt)

		t.lock.Lock()
		t.err = err
		t.isComplete = true
		t.lock.Unlock()

		if complete != nil {
			complete <- err
		}

		if err == nil {
			logger.LogDepthWithFields(ctx, logger.LevelVerbose, caller, nil,
				"Finished: %s (thread)", name)
		} else if errors.Cause(err) == Interrupted {
			logger.LogDepthWithFields(ctx, logger.LevelVerbose, caller, nil,
				"Finished: %s (thread) : %s", name, err)
		} else {
			logger.LogDepthWithFields(ctx, logger.LevelWarn, caller, nil,
				"Finished: %s (thread): %s", name, err)
		}
	}()
}

func (t *InterruptableThread) Stop(ctx context.Context) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.interrupt != nil {
		close(t.interrupt)
		t.interrupt = nil
	}
}

func (t *InterruptableThread) IsComplete() bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.isComplete
}

func (t *InterruptableThread) Error() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.err
}
