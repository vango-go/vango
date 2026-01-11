package vango

import "testing"

func TestNewSharedMemo_PerSessionIsolationAndReactivity(t *testing.T) {
	count := NewSharedSignal(0)
	doubled := NewSharedMemo(func() int {
		return count.Get() * 2
	})

	// Session A
	ownerA := NewOwner(nil)
	storeA := NewSimpleSessionSignalStore()

	oldOwner := setCurrentOwner(ownerA)
	defer setCurrentOwner(oldOwner)
	SetContext(SessionSignalStoreKey, storeA)

	if got := doubled.Get(); got != 0 {
		t.Fatalf("session A initial doubled.Get() = %d, want 0", got)
	}

	memoA := doubled.Memo()
	if memoA == nil {
		t.Fatal("session A doubled.Memo() = nil, want non-nil")
	}

	count.Set(2)
	if got := doubled.Get(); got != 4 {
		t.Fatalf("session A after Set(2) doubled.Get() = %d, want 4", got)
	}

	// Session B (different store/owner)
	ownerB := NewOwner(nil)
	storeB := NewSimpleSessionSignalStore()

	setCurrentOwner(ownerB)
	SetContext(SessionSignalStoreKey, storeB)

	if got := doubled.Get(); got != 0 {
		t.Fatalf("session B initial doubled.Get() = %d, want 0", got)
	}

	memoB := doubled.Memo()
	if memoB == nil {
		t.Fatal("session B doubled.Memo() = nil, want non-nil")
	}
	if memoA == memoB {
		t.Fatal("shared memo instance was reused across sessions; want per-session memo instances")
	}

	count.Set(3)
	if got := doubled.Get(); got != 6 {
		t.Fatalf("session B after Set(3) doubled.Get() = %d, want 6", got)
	}

	// Back to session A: should be unaffected by session B writes.
	setCurrentOwner(ownerA)
	SetContext(SessionSignalStoreKey, storeA)

	if got := doubled.Get(); got != 4 {
		t.Fatalf("session A after session B writes doubled.Get() = %d, want 4", got)
	}

	count.Set(5)
	if got := doubled.Get(); got != 10 {
		t.Fatalf("session A after Set(5) doubled.Get() = %d, want 10", got)
	}

	// Owner restored by defer.
}
