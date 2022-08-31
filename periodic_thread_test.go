package threads

import (
	"context"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/tokenized/logger"
)

func Test_PeriodicThread_Complete(t *testing.T) {
	ctx := context.Background()

	t.Logf("Starting")
	defer t.Logf("Completed")
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("Top level panic : %s : %s", err, string(debug.Stack()))
		}
	}()

	thread := NewPeriodicThread("Test Thread", periodicFunction, time.Millisecond*100)

	var wait sync.WaitGroup
	thread.SetWait(&wait)
	complete := thread.GetCompleteChannel()

	thread.Start(ctx)

	select {
	case <-complete:
		t.Errorf("Thread completed")

	case <-time.After(time.Millisecond * 200):
		t.Logf("Thread correctly did not complete")
	}

	thread.Stop(ctx)

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

func periodicFunction(ctx context.Context) error {
	logger.Info(ctx, "Periodic executed")
	return nil
}
