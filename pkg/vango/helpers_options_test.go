package vango

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestOptionInterfaces_AreCallable(t *testing.T) {
	CancelLatest().isActionOption()
	Queue(1).isActionOption()

	AllowWrites().isEffectOption()
	EffectTxName("x").isEffectOption()

	IntervalImmediate().isIntervalOption()
	IntervalTxName("x").isIntervalOption()

	SubscribeTxName("x").isSubscribeOption()
	GoLatestTxName("x").isGoLatestOption()
	GoLatestForceRestart().isGoLatestOption()
	TimeoutTxName("x").isTimeoutOption()
}

func TestSubscribeTxName_WrapsDispatchTx(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	stream := &mockStream[string]{}
	var gotMu sync.Mutex
	var got []string

	cleanup := Subscribe(stream, func(msg string) {
		gotMu.Lock()
		got = append(got, msg)
		gotMu.Unlock()
	}, SubscribeTxName("stream"))
	defer cleanup()

	stream.emit("a")
	stream.emit("b")
	time.Sleep(5 * time.Millisecond)

	gotMu.Lock()
	defer gotMu.Unlock()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("received = %v, want %v", got, []string{"a", "b"})
	}
}

func TestGoLatestForceRestart_StartsNewWorkOnSameKey(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	var workCount int
	var applyCount int
	var mu sync.Mutex

	e := CreateEffect(func() Cleanup {
		return GoLatest("same",
			func(ctx context.Context, key string) (int, error) {
				mu.Lock()
				workCount++
				n := workCount
				mu.Unlock()
				return n, nil
			},
			func(r int, err error) {
				mu.Lock()
				applyCount++
				mu.Unlock()
			},
			GoLatestForceRestart(),
			GoLatestTxName("force"),
		)
	}, EffectTxName("effect"))

	time.Sleep(10 * time.Millisecond)

	// Re-run effect with same key; force restart should start work again.
	e.pending.Store(true)
	e.run()

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if workCount < 2 || applyCount < 2 {
		t.Fatalf("workCount=%d applyCount=%d, want >=2 each", workCount, applyCount)
	}
}

func TestTimeoutTxName_OptionIsApplied(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	done := make(chan struct{})
	cleanup := Timeout(1*time.Millisecond, func() { close(done) }, TimeoutTxName("named"))
	defer cleanup()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("timeout did not fire")
	}
}
