package vango

import (
	"testing"
)

// Benchmark tests for the reactive system.
// Target performance:
// - Signal.Get() (no tracking): < 10 ns
// - Signal.Get() (with tracking): < 50 ns
// - Signal.Set() (10 subscribers): < 200 ns
// - Memo.Get() (cached): < 15 ns
// - Batch (100 updates): < 5 µs

func BenchmarkSignalGetNoTracking(b *testing.B) {
	s := NewSignal(42)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = s.Get()
	}
}

func BenchmarkSignalGetWithTracking(b *testing.B) {
	s := NewSignal(42)
	listener := newTestListener()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		WithListener(listener, func() {
			_ = s.Get()
		})
	}
}

func BenchmarkSignalPeek(b *testing.B) {
	s := NewSignal(42)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = s.Peek()
	}
}

func BenchmarkSignalSetNoSubscribers(b *testing.B) {
	s := NewSignal(0)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Set(i)
	}
}

func BenchmarkSignalSet1Subscriber(b *testing.B) {
	s := NewSignal(0)
	listener := newTestListener()
	WithListener(listener, func() {
		_ = s.Get()
	})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Set(i)
	}
}

func BenchmarkSignalSet10Subscribers(b *testing.B) {
	s := NewSignal(0)

	for i := 0; i < 10; i++ {
		listener := newTestListener()
		WithListener(listener, func() {
			_ = s.Get()
		})
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Set(i)
	}
}

func BenchmarkSignalSet100Subscribers(b *testing.B) {
	s := NewSignal(0)

	for i := 0; i < 100; i++ {
		listener := newTestListener()
		WithListener(listener, func() {
			_ = s.Get()
		})
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Set(i)
	}
}

func BenchmarkSignalUpdate(b *testing.B) {
	s := NewSignal(0)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Update(func(n int) int { return n + 1 })
	}
}

func BenchmarkMemoGetCached(b *testing.B) {
	count := NewSignal(42)
	m := NewMemo(func() int { return count.Get() * 2 })

	// Prime the cache
	_ = m.Get()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = m.Get()
	}
}

func BenchmarkMemoRecompute(b *testing.B) {
	count := NewSignal(0)
	m := NewMemo(func() int { return count.Get() * 2 })

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		count.Set(i)
		_ = m.Get()
	}
}

func BenchmarkMemoChain3(b *testing.B) {
	a := NewSignal(0)
	b1 := NewMemo(func() int { return a.Get() * 2 })
	c := NewMemo(func() int { return b1.Get() * 2 })
	d := NewMemo(func() int { return c.Get() * 2 })

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Set(i)
		_ = d.Get()
	}
}

func BenchmarkMemoChain5(b *testing.B) {
	a := NewSignal(0)
	b1 := NewMemo(func() int { return a.Get() * 2 })
	c := NewMemo(func() int { return b1.Get() * 2 })
	d := NewMemo(func() int { return c.Get() * 2 })
	e := NewMemo(func() int { return d.Get() * 2 })
	f := NewMemo(func() int { return e.Get() * 2 })

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Set(i)
		_ = f.Get()
	}
}

func BenchmarkBatch10Updates(b *testing.B) {
	signals := make([]*Signal[int], 10)
	for i := range signals {
		signals[i] = NewSignal(0)
	}

	listener := newTestListener()
	WithListener(listener, func() {
		for _, s := range signals {
			_ = s.Get()
		}
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Batch(func() {
			for j, s := range signals {
				s.Set(i*10 + j)
			}
		})
	}
}

func BenchmarkBatch100Updates(b *testing.B) {
	signals := make([]*Signal[int], 100)
	for i := range signals {
		signals[i] = NewSignal(0)
	}

	listener := newTestListener()
	WithListener(listener, func() {
		for _, s := range signals {
			_ = s.Get()
		}
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Batch(func() {
			for j, s := range signals {
				s.Set(i*100 + j)
			}
		})
	}
}

func BenchmarkEffectCreation(b *testing.B) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	count := NewSignal(0)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		WithOwner(owner, func() {
			CreateEffect(func() Cleanup {
				_ = count.Get()
				return nil
			})
		})
	}
}

func BenchmarkEffectRun(b *testing.B) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	count := NewSignal(0)

	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			_ = count.Get()
			return nil
		})
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		count.Set(i)
		owner.RunPendingEffects(nil)
	}
}

func BenchmarkIntSignalInc(b *testing.B) {
	s := NewIntSignal(0)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Inc()
	}
}

func BenchmarkSliceSignalAppend(b *testing.B) {
	s := NewSliceSignal([]int{})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Append(i)
	}
}

func BenchmarkMapSignalSetKey(b *testing.B) {
	s := NewMapSignal[string, int](nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.SetKey("key", i)
	}
}

func BenchmarkGetTrackingContext(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = getTrackingContext()
	}
}

func BenchmarkWithListener(b *testing.B) {
	listener := newTestListener()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		WithListener(listener, func() {})
	}
}

// BenchmarkRealisticComponent simulates a realistic component with:
// - 5 signals
// - 3 derived memos
// - 1 effect
// - User interactions causing updates
func BenchmarkRealisticComponent(b *testing.B) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	// Signals
	firstName := NewSignal("John")
	lastName := NewSignal("Doe")
	age := NewSignal(30)
	email := NewSignal("john@example.com")
	active := NewBoolSignal(true)

	// Derived
	fullName := NewMemo(func() string {
		return firstName.Get() + " " + lastName.Get()
	})
	isAdult := NewMemo(func() bool {
		return age.Get() >= 18
	})
	canContact := NewMemo(func() bool {
		return active.Get() && len(email.Get()) > 0
	})

	// Effect
	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			_ = fullName.Get()
			_ = isAdult.Get()
			_ = canContact.Get()
			return nil
		})
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate user interaction
		Batch(func() {
			firstName.Set("Jane")
			lastName.Set("Smith")
		})
		owner.RunPendingEffects(nil)

		age.Set(25)
		owner.RunPendingEffects(nil)

		active.Toggle()
		owner.RunPendingEffects(nil)

		// Read derived values
		_ = fullName.Get()
		_ = isAdult.Get()
		_ = canContact.Get()
	}
}
