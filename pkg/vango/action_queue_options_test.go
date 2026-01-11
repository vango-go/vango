package vango

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestActionCancelLatestOption_CancelsInFlight(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	started1 := make(chan struct{})
	cancelled1 := make(chan struct{})
	started2 := make(chan struct{})
	finish2 := make(chan struct{})

	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		switch arg {
		case 1:
			close(started1)
			<-ctx.Done()
			close(cancelled1)
			return 0, ctx.Err()
		case 2:
			close(started2)
			<-finish2
			return 2, nil
		default:
			return arg, nil
		}
	}, CancelLatest())

	if !action.Run(1) {
		t.Fatalf("first Run should be accepted")
	}
	<-started1

	if !action.Run(2) {
		t.Fatalf("second Run should be accepted with CancelLatest")
	}
	<-started2
	<-cancelled1

	close(finish2)
	time.Sleep(10 * time.Millisecond)

	if !action.IsSuccess() {
		t.Fatalf("expected final state success, got %v", action.State())
	}
	if got, ok := action.Result(); !ok || got != 2 {
		t.Fatalf("Result() = (%v, %v), want (%v, true)", got, ok, 2)
	}
}

func TestActionQueue_PreservesOrder_QueueFullAndCallbacks(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	started := make(chan int, 10)
	permits := map[int]chan struct{}{
		1: make(chan struct{}),
		2: make(chan struct{}),
		3: make(chan struct{}),
	}

	var onStartCount atomic.Int32
	var onSuccessMu sync.Mutex
	var onSuccess []int

	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		started <- arg
		<-permits[arg]
		return arg, nil
	},
		Queue(1),
		ActionTxName("test:queue"),
		OnActionStart(func() { onStartCount.Add(1) }),
		OnActionSuccess(func(v any) {
			onSuccessMu.Lock()
			onSuccess = append(onSuccess, v.(int))
			onSuccessMu.Unlock()
		}),
	)

	if !action.Run(1) {
		t.Fatalf("Run(1) should be accepted")
	}
	if got := <-started; got != 1 {
		t.Fatalf("first started = %d, want %d", got, 1)
	}

	if !action.Run(2) {
		t.Fatalf("Run(2) should be accepted (queued)")
	}
	if action.Run(3) {
		t.Fatalf("Run(3) should be rejected when queue is full")
	}
	if action.Error() != ErrQueueFull {
		t.Fatalf("Error() = %v, want %v", action.Error(), ErrQueueFull)
	}
	if action.State() != ActionError {
		t.Fatalf("State() after queue-full rejection = %v, want %v", action.State(), ActionError)
	}

	close(permits[1])
	if got := <-started; got != 2 {
		t.Fatalf("second started = %d, want %d", got, 2)
	}
	close(permits[2])
	time.Sleep(10 * time.Millisecond)

	onSuccessMu.Lock()
	defer onSuccessMu.Unlock()
	if len(onSuccess) != 2 || onSuccess[0] != 1 || onSuccess[1] != 2 {
		t.Fatalf("OnActionSuccess results = %v, want %v", onSuccess, []int{1, 2})
	}
	if onStartCount.Load() < 2 {
		t.Fatalf("OnActionStart count = %d, want >= 2", onStartCount.Load())
	}
}

func TestActionOnErrorCallback_FiresOnWorkError(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	var onErrorCount atomic.Int32
	expected := ErrActionRunning // any sentinel error is fine here

	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		return 0, expected
	}, OnActionError(func(err error) {
		if err == expected {
			onErrorCount.Add(1)
		}
	}))

	if !action.Run(1) {
		t.Fatalf("Run should be accepted")
	}
	time.Sleep(10 * time.Millisecond)

	if action.State() != ActionError {
		t.Fatalf("State() = %v, want %v", action.State(), ActionError)
	}
	if action.Error() != expected {
		t.Fatalf("Error() = %v, want %v", action.Error(), expected)
	}
	if onErrorCount.Load() != 1 {
		t.Fatalf("OnActionError count = %d, want %d", onErrorCount.Load(), 1)
	}
}

func TestActionIsRunning(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	started := make(chan struct{})
	finish := make(chan struct{})

	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		close(started)
		<-finish
		return arg, nil
	})

	if !action.Run(1) {
		t.Fatalf("Run should be accepted")
	}
	<-started
	if !action.IsRunning() {
		t.Fatalf("IsRunning() should be true while work is in flight")
	}
	close(finish)
	time.Sleep(10 * time.Millisecond)
}
