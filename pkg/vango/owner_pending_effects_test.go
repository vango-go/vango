package vango

import "testing"

func TestOwnerHasPendingEffects_RecursesIntoChildren(t *testing.T) {
	root := NewOwner(nil)
	defer root.Dispose()
	child := NewOwner(root)
	defer child.Dispose()

	count := NewSignal(0)

	WithOwner(child, func() {
		CreateEffect(func() Cleanup {
			_ = count.Get()
			return nil
		})
	})

	if root.HasPendingEffects() {
		t.Fatalf("HasPendingEffects() should be false when nothing is scheduled")
	}

	count.Set(1) // schedules effect on child owner
	if !root.HasPendingEffects() {
		t.Fatalf("HasPendingEffects() should be true after dependency change")
	}

	root.RunPendingEffects(nil)
	if root.HasPendingEffects() {
		t.Fatalf("HasPendingEffects() should be false after running pending effects")
	}
}

