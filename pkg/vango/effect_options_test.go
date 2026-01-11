package vango

import "testing"

func TestEffectStrictMode_PanicsWithoutAllowWrites_AllowsWithOption(t *testing.T) {
	old := EffectStrictMode
	EffectStrictMode = StrictEffectPanic
	defer func() { EffectStrictMode = old }()

	owner := NewOwner(nil)
	defer owner.Dispose()

	s := NewSignal(0)

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for effect-time write without AllowWrites")
			}
		}()
		WithOwner(owner, func() {
			CreateEffect(func() Cleanup {
				s.Set(1)
				return nil
			})
		})
	}()

	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			s.Set(2)
			return nil
		}, AllowWrites(), EffectTxName("init"))
	})
	if got := s.Get(); got != 2 {
		t.Fatalf("signal value after AllowWrites effect = %d, want %d", got, 2)
	}
}

