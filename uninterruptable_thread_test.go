package threads

import (
	"context"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/tokenized/logger"
)

func Test_UninterupptableThread_Complete(t *testing.T) {
	ctx := context.Background()

	t.Logf("Starting")
	defer t.Logf("Completed")
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("Top level panic : %s : %s", err, string(debug.Stack()))
		}
	}()

	incoming := make(chan string)
	thread := NewUninterruptableThread("Test Thread", func(ctx context.Context) error {
		return channelReadFunction(ctx, incoming)
	})

	var wait sync.WaitGroup
	thread.SetWait(&wait)
	complete := thread.GetCompleteChannel()

	thread.Start(ctx)

	incoming <- "value 1"

	select {
	case <-complete:
		t.Errorf("Thread completed")

	case <-time.After(time.Millisecond * 200):
		t.Logf("Thread correctly did not complete")
	}

	incoming <- "value 2"

	close(incoming)

	select {
	case <-complete:
		t.Logf("Thread completed")

	case <-time.After(time.Millisecond * 50):
		t.Errorf("Thread should have stopped")
	}

	wait.Wait()
	threadErr := thread.Error()
	if threadErr != nil {
		t.Errorf("Thread should not return error : %s", threadErr)
	}
}

func channelReadFunction(ctx context.Context, incoming <-chan string) error {
	for s := range incoming {
		logger.Info(ctx, "Received : %s", s)
	}

	return nil
}
