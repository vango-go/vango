package vango

import "testing"

func TestBatchSingleNotification(t *testing.T) {
	a := NewSignal(0)
	b := NewSignal(0)
	c := NewSignal(0)

	listener := newTestListener()

	// Subscribe to all signals
	WithListener(listener, func() {
		_ = a.Get()
		_ = b.Get()
		_ = c.Get()
	})

	// Without batch, each Set would notify
	Batch(func() {
		a.Set(1)
		b.Set(2)
		c.Set(3)
	})

	// Should only notify once (deduplicated)
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification (batched), got %d", listener.getDirtyCount())
	}
}

func TestBatchDeduplication(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	// Multiple updates to same signal in batch
	Batch(func() {
		count.Set(1)
		count.Set(2)
		count.Set(3)
		count.Set(4)
		count.Set(5)
	})

	// Should only notify once
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification (deduplicated), got %d", listener.getDirtyCount())
	}

	// Value should be final value
	if count.Get() != 5 {
		t.Errorf("expected final value 5, got %d", count.Get())
	}
}

func TestBatchNested(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	Batch(func() {
		count.Set(1)

		Batch(func() {
			count.Set(2)

			Batch(func() {
				count.Set(3)
			})

			// Still inside outer batch, no notification yet
			if listener.getDirtyCount() != 0 {
				t.Errorf("inner batch should not notify, got %d", listener.getDirtyCount())
			}
		})

		// Still inside outer batch
		if listener.getDirtyCount() != 0 {
			t.Errorf("middle batch should not notify, got %d", listener.getDirtyCount())
		}
	})

	// Only now should we notify
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification after all batches, got %d", listener.getDirtyCount())
	}
}

func TestBatchMultipleListeners(t *testing.T) {
	a := NewSignal(0)
	b := NewSignal(0)

	listener1 := newTestListener()
	listener2 := newTestListener()
	listener3 := newTestListener()

	// Subscribe listeners to different signals
	WithListener(listener1, func() {
		_ = a.Get()
	})
	WithListener(listener2, func() {
		_ = b.Get()
	})
	WithListener(listener3, func() {
		_ = a.Get()
		_ = b.Get()
	})

	Batch(func() {
		a.Set(1)
		b.Set(2)
	})

	// All three should be notified exactly once
	if listener1.getDirtyCount() != 1 {
		t.Errorf("listener1 expected 1 notification, got %d", listener1.getDirtyCount())
	}
	if listener2.getDirtyCount() != 1 {
		t.Errorf("listener2 expected 1 notification, got %d", listener2.getDirtyCount())
	}
	if listener3.getDirtyCount() != 1 {
		t.Errorf("listener3 expected 1 notification, got %d", listener3.getDirtyCount())
	}
}

func TestBatchNoChanges(t *testing.T) {
	count := NewSignal(5)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	// Batch with no actual changes
	Batch(func() {
		count.Set(5) // Same value
		count.Set(5) // Same value again
	})

	// No notifications (value didn't change)
	if listener.getDirtyCount() != 0 {
		t.Errorf("expected 0 notifications for unchanged value, got %d", listener.getDirtyCount())
	}
}

func TestBatchWithPanic(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to propagate")
		}

		// The batch's deferred cleanup still runs, processing pending updates
		// So we get 1 notification from the batch cleanup
		// Then setting to 100 causes another notification
		count.Set(100)
		// Total: 1 (from batch cleanup) + 1 (from this Set) = 2
		if listener.getDirtyCount() != 2 {
			t.Errorf("expected 2 notifications after recovery, got %d", listener.getDirtyCount())
		}
	}()

	Batch(func() {
		count.Set(1)
		panic("test panic")
	})
}

func TestUntracked(t *testing.T) {
	count := NewSignal(42)
	listener := newTestListener()

	// Read within listener context but inside Untracked
	WithListener(listener, func() {
		Untracked(func() {
			value := count.Get()
			if value != 42 {
				t.Errorf("expected 42, got %d", value)
			}
		})
	})

	// Should not be subscribed
	count.Set(100)
	if listener.getDirtyCount() != 0 {
		t.Errorf("Untracked should prevent subscription, got %d notifications", listener.getDirtyCount())
	}
}

func TestUntrackedNested(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get() // This subscribes

		Untracked(func() {
			// This read doesn't subscribe
			other := NewSignal(999)
			_ = other.Get()
		})

		// Back to tracking context
		// (already subscribed from first Get)
	})

	// Should have subscribed from the first Get
	count.Set(1)
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification, got %d", listener.getDirtyCount())
	}
}

func TestUntrackedRestoresContext(t *testing.T) {
	outer := NewSignal(0)
	inner := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = outer.Get() // Subscribe

		Untracked(func() {
			_ = inner.Get() // No subscribe
		})

		// Should still be able to track after Untracked
		// (listener context was restored)
	})

	// Only outer should notify
	inner.Set(1)
	if listener.getDirtyCount() != 0 {
		t.Errorf("inner signal should not notify, got %d", listener.getDirtyCount())
	}

	outer.Set(1)
	if listener.getDirtyCount() != 1 {
		t.Errorf("outer signal should notify, got %d", listener.getDirtyCount())
	}
}

func TestUntrackedGet(t *testing.T) {
	count := NewSignal(42)

	// UntrackedGet should work same as Peek
	value := UntrackedGet(count)
	if value != 42 {
		t.Errorf("expected 42, got %d", value)
	}

	// Should not subscribe even if called within listener context
	listener := newTestListener()
	WithListener(listener, func() {
		_ = UntrackedGet(count)
	})

	count.Set(100)
	if listener.getDirtyCount() != 0 {
		t.Errorf("UntrackedGet should not subscribe, got %d notifications", listener.getDirtyCount())
	}
}

func TestBatchWithMemo(t *testing.T) {
	a := NewSignal(1)
	b := NewSignal(2)

	sum := NewMemo(func() int {
		return a.Get() + b.Get()
	})

	listener := newTestListener()
	WithListener(listener, func() {
		_ = sum.Get()
	})

	// Initial: 1 + 2 = 3
	if sum.Get() != 3 {
		t.Errorf("expected 3, got %d", sum.Get())
	}

	// Batch updates
	Batch(func() {
		a.Set(10)
		b.Set(20)
	})

	// Memo should be notified once (deduplicated from a and b)
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification, got %d", listener.getDirtyCount())
	}

	// Final value: 10 + 20 = 30
	if sum.Get() != 30 {
		t.Errorf("expected 30, got %d", sum.Get())
	}
}

// =============================================================================
// Tx() and TxNamed() Tests (Phase 4 additions)
// =============================================================================

func TestTxIsAliasForBatch(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	// Tx should behave exactly like Batch
	Tx(func() {
		count.Set(1)
		count.Set(2)
		count.Set(3)
	})

	// Only one notification (batched)
	if listener.getDirtyCount() != 1 {
		t.Errorf("Tx: expected 1 notification, got %d", listener.getDirtyCount())
	}

	if count.Get() != 3 {
		t.Errorf("Tx: expected final value 3, got %d", count.Get())
	}
}

func TestTxNested(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	Tx(func() {
		count.Set(1)
		Tx(func() {
			count.Set(2)
		})
		count.Set(3)
	})

	// Only one notification after outermost Tx
	if listener.getDirtyCount() != 1 {
		t.Errorf("Nested Tx: expected 1 notification, got %d", listener.getDirtyCount())
	}
}

func TestTxWithBatchNested(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	// Tx and Batch can be nested interchangeably
	Tx(func() {
		count.Set(1)
		Batch(func() {
			count.Set(2)
		})
		count.Set(3)
	})

	if listener.getDirtyCount() != 1 {
		t.Errorf("Tx+Batch: expected 1 notification, got %d", listener.getDirtyCount())
	}
}

func TestTxNamed(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	TxNamed("test-transaction", func() {
		count.Set(1)
		count.Set(2)
		count.Set(3)
	})

	// Same behavior as Tx
	if listener.getDirtyCount() != 1 {
		t.Errorf("TxNamed: expected 1 notification, got %d", listener.getDirtyCount())
	}

	if count.Get() != 3 {
		t.Errorf("TxNamed: expected final value 3, got %d", count.Get())
	}
}

func TestTxNamedDebugMode(t *testing.T) {
	oldDebug := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebug }()

	count := NewSignal(0)

	// Should not panic and should still work
	TxNamed("debug-transaction", func() {
		count.Set(42)
	})

	if count.Get() != 42 {
		t.Errorf("TxNamed debug: expected 42, got %d", count.Get())
	}
}

func TestTxNamedNested(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	TxNamed("outer", func() {
		count.Set(1)
		TxNamed("inner", func() {
			count.Set(2)
		})
		count.Set(3)
	})

	if listener.getDirtyCount() != 1 {
		t.Errorf("Nested TxNamed: expected 1 notification, got %d", listener.getDirtyCount())
	}
}

func TestTxMultipleSignals(t *testing.T) {
	a := NewSignal(0)
	b := NewSignal(0)
	c := NewSignal(0)

	listener := newTestListener()
	WithListener(listener, func() {
		_ = a.Get()
		_ = b.Get()
		_ = c.Get()
	})

	Tx(func() {
		a.Set(1)
		b.Set(2)
		c.Set(3)
	})

	// Single notification even with multiple signals
	if listener.getDirtyCount() != 1 {
		t.Errorf("Tx multi-signal: expected 1 notification, got %d", listener.getDirtyCount())
	}
}

func TestTxPanicRecovery(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to propagate")
		}
		// Verify transaction cleanup happened via defer
	}()

	Tx(func() {
		count.Set(1)
		panic("test panic in transaction")
	})
}

func TestTxNamedPanicRecovery(t *testing.T) {
	count := NewSignal(0)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to propagate")
		}
	}()

	TxNamed("panic-transaction", func() {
		count.Set(1)
		panic("test panic in named transaction")
	})
}
