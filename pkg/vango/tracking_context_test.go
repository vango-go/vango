package vango

import "testing"

func TestWithCtx_SetsUseCtxAndRestores(t *testing.T) {
	setCurrentCtx(nil)
	if UseCtx() != nil {
		t.Fatalf("UseCtx() should be nil without WithCtx")
	}

	mockC := newMockCtx()
	WithCtx(mockC, func() {
		if got := UseCtx(); got != mockC {
			t.Fatalf("UseCtx() = %T, want %T", got, mockC)
		}
	})

	if UseCtx() != nil {
		t.Fatalf("UseCtx() should be restored to nil after WithCtx")
	}
}

func TestTrackingContext_SetTrackingContextAndEffectCallSiteIdx(t *testing.T) {
	// Explicitly set a context then clear it, to cover setTrackingContext.
	ctx := &TrackingContext{batchDepth: 123}
	setTrackingContext(ctx)
	if got := getTrackingContext(); got.batchDepth != 123 {
		t.Fatalf("getTrackingContext() batchDepth = %d, want %d", got.batchDepth, 123)
	}
	setTrackingContext(nil)
	if got := getTrackingContext(); got.batchDepth != 0 {
		t.Fatalf("expected cleared tracking context; batchDepth=%d", got.batchDepth)
	}

	owner := NewOwner(nil)
	defer owner.Dispose()

	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			if got := getEffectCallSiteIdx(); got != 0 {
				t.Fatalf("initial call-site idx = %d, want %d", got, 0)
			}
			if idx := incrementEffectCallSiteIdx(); idx != 0 {
				t.Fatalf("incrementEffectCallSiteIdx() = %d, want %d", idx, 0)
			}
			if got := getEffectCallSiteIdx(); got != 1 {
				t.Fatalf("call-site idx after increment = %d, want %d", got, 1)
			}
			return nil
		}, AllowWrites())
	})
}

