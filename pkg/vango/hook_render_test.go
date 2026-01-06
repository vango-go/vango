package vango_test

import (
	"testing"

	"github.com/vango-go/vango/pkg/features/form"
	"github.com/vango-go/vango/pkg/vango"
)

type hookFormTest struct {
	Name string
}

func TestRenderHookSlotStability(t *testing.T) {
	owner := vango.NewOwner(nil)
	defer owner.Dispose()

	var sig1, sig2 *vango.Signal[int]
	var memo1, memo2 *vango.Memo[int]
	var ref1, ref2 *vango.Ref[int]
	var form1, form2 *form.Form[hookFormTest]
	var eff1, eff2 *vango.Effect

	runs := 0

	render := func(initial int) {
		owner.StartRender()
		sig := vango.NewSignal(initial)
		memo := vango.NewMemo(func() int { return sig.Get() })
		ref := vango.NewRef[int](0)
		frm := form.UseForm(hookFormTest{})
		eff := vango.CreateEffect(func() vango.Cleanup {
			runs++
			_ = memo.Get()
			return nil
		})
		owner.EndRender()

		if sig1 == nil {
			sig1, memo1, ref1, form1, eff1 = sig, memo, ref, frm, eff
		} else {
			sig2, memo2, ref2, form2, eff2 = sig, memo, ref, frm, eff
		}
	}

	vango.WithOwner(owner, func() {
		render(1)
	})

	if runs != 0 {
		t.Fatalf("effect ran during render, runs=%d", runs)
	}

	owner.RunPendingEffects()
	if runs != 1 {
		t.Fatalf("expected 1 effect run after commit, got %d", runs)
	}

	vango.WithOwner(owner, func() {
		render(999)
	})

	if sig1 != sig2 {
		t.Error("signal did not persist across renders")
	}
	if sig2.Get() != 1 {
		t.Errorf("signal reinitialized on rerender, got %d want %d", sig2.Get(), 1)
	}
	if memo1 != memo2 {
		t.Error("memo did not persist across renders")
	}
	if ref1 != ref2 {
		t.Error("ref did not persist across renders")
	}
	if form1 != form2 {
		t.Error("form did not persist across renders")
	}
	if eff1 != eff2 {
		t.Error("effect did not persist across renders")
	}
}

func TestEffectDeferredUntilAfterRender(t *testing.T) {
	owner := vango.NewOwner(nil)
	defer owner.Dispose()

	runs := 0
	vango.WithOwner(owner, func() {
		owner.StartRender()
		vango.CreateEffect(func() vango.Cleanup {
			runs++
			return nil
		})
		owner.EndRender()
	})

	if runs != 0 {
		t.Fatalf("effect ran during render, runs=%d", runs)
	}

	owner.RunPendingEffects()
	if runs != 1 {
		t.Fatalf("expected 1 effect run after commit, got %d", runs)
	}
}

func TestRunPendingEffectsRecursive(t *testing.T) {
	root := vango.NewOwner(nil)
	defer root.Dispose()

	child := vango.NewOwner(root)

	runs := 0
	vango.WithOwner(child, func() {
		child.StartRender()
		vango.CreateEffect(func() vango.Cleanup {
			runs++
			return nil
		})
		child.EndRender()
	})

	if runs != 0 {
		t.Fatalf("effect ran during render, runs=%d", runs)
	}

	root.RunPendingEffects()
	if runs != 1 {
		t.Fatalf("expected child effect to run from root RunPendingEffects, got %d", runs)
	}
}
