package threads

import (
	"context"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func Test_InterupptableThread_Complete(t *testing.T) {
	ctx := context.Background()

	t.Logf("Starting")
	defer t.Logf("Completed")
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("Top level panic : %s : %s", err, string(debug.Stack()))
		}
	}()

	thread := NewInterruptableThread("Test Thread", interruptableFunctionThatWaits)

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

func Test_InterupptableThread_Error(t *testing.T) {
	ctx := context.Background()

	t.Logf("Starting")
	defer t.Logf("Completed")
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("Top level panic : %s : %s", err, string(debug.Stack()))
		}
	}()

	thread := NewInterruptableThread("Test Thread", interruptableFunctionThatErrors)

	var wait sync.WaitGroup
	thread.SetWait(&wait)
	complete := thread.GetCompleteChannel()

	thread.Start(ctx)

	select {
	case err := <-complete:
		t.Logf("Thread complete")
		if err == nil {
			t.Errorf("Complete channel should return error")
		} else {
			t.Logf("Complete channel error : %s", err)
		}

	case <-time.After(time.Millisecond * 1100):
		t.Errorf("Timed out")
	}

	wait.Wait()
	threadErr := thread.Error()
	if threadErr != nil {
		t.Logf("Thread error : %s", threadErr)
	} else {
		t.Errorf("Thread should return error")
	}
}

func Test_InterupptableThread_Panic(t *testing.T) {
	ctx := context.Background()

	t.Logf("Starting")
	defer t.Logf("Completed")
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("Top level panic : %s : %s", err, string(debug.Stack()))
		}
	}()

	thread := NewInterruptableThread("Test Thread", interruptableFunctionThatPanics)

	var wait sync.WaitGroup
	thread.SetWait(&wait)
	complete := thread.GetCompleteChannel()

	thread.Start(ctx)

	select {
	case err := <-complete:
		t.Logf("Thread complete")
		if err == nil {
			t.Errorf("Complete channel should return error")
		} else {
			t.Logf("Thread error : %s", err)
		}

	case <-time.After(time.Millisecond * 1100):
		t.Errorf("Timed out")
	}

	wait.Wait()
	threadErr := thread.Error()
	if threadErr == nil {
		t.Errorf("Thread should return error")
	} else {
		t.Logf("Thread error : %s", threadErr)
	}
}

func interruptableFunctionThatWaits(ctx context.Context, interrupt <-chan interface{}) error {
	select {
	case <-interrupt:
		return nil

	case <-time.After(time.Second * 10):
		return errors.New("Thread timed out")
	}

	return nil
}

func interruptableFunctionThatPanics(ctx context.Context, interrupt <-chan interface{}) error {
	time.Sleep(time.Millisecond * 100)
	panic("test panic")
	time.Sleep(time.Second * 5)

	return nil
}

func interruptableFunctionThatErrors(ctx context.Context, interrupt <-chan interface{}) error {
	time.Sleep(time.Millisecond * 100)
	return errors.New("Failure")
}
