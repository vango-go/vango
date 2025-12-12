package protocol

import (
	"testing"
)

func TestPatchEncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		patch Patch
	}{
		{
			name:  "set_text",
			patch: NewSetTextPatch("h1", "Hello, World!"),
		},
		{
			name:  "set_attr",
			patch: NewSetAttrPatch("h2", "class", "active highlighted"),
		},
		{
			name:  "remove_attr",
			patch: NewRemoveAttrPatch("h3", "disabled"),
		},
		{
			name: "insert_node",
			patch: NewInsertNodePatch("h4", "h0", 2, &VNodeWire{
				Kind: 0, // Element
				Tag:  "div",
				HID:  "h4",
				Attrs: map[string]string{
					"class": "new-item",
				},
				Children: []*VNodeWire{
					NewTextWire("New content"),
				},
			}),
		},
		{
			name:  "remove_node",
			patch: NewRemoveNodePatch("h5"),
		},
		{
			name:  "move_node",
			patch: NewMoveNodePatch("h6", "h0", 5),
		},
		{
			name: "replace_node",
			patch: NewReplaceNodePatch("h7", &VNodeWire{
				Kind: 0,
				Tag:  "span",
				HID:  "h7",
				Attrs: map[string]string{
					"id": "replacement",
				},
			}),
		},
		{
			name:  "set_value",
			patch: NewSetValuePatch("h8", "new input value"),
		},
		{
			name:  "set_checked_true",
			patch: NewSetCheckedPatch("h9", true),
		},
		{
			name:  "set_checked_false",
			patch: NewSetCheckedPatch("h10", false),
		},
		{
			name:  "set_selected_true",
			patch: NewSetSelectedPatch("h11", true),
		},
		{
			name:  "focus",
			patch: NewFocusPatch("h12"),
		},
		{
			name:  "blur",
			patch: NewBlurPatch("h13"),
		},
		{
			name:  "scroll_to_instant",
			patch: NewScrollToPatch("h14", 100, 200, ScrollInstant),
		},
		{
			name:  "scroll_to_smooth",
			patch: NewScrollToPatch("h15", 0, 500, ScrollSmooth),
		},
		{
			name:  "add_class",
			patch: NewAddClassPatch("h16", "highlight"),
		},
		{
			name:  "remove_class",
			patch: NewRemoveClassPatch("h17", "hidden"),
		},
		{
			name:  "toggle_class",
			patch: NewToggleClassPatch("h18", "active"),
		},
		{
			name:  "set_style",
			patch: NewSetStylePatch("h19", "background-color", "red"),
		},
		{
			name:  "remove_style",
			patch: NewRemoveStylePatch("h20", "display"),
		},
		{
			name:  "set_data",
			patch: NewSetDataPatch("h21", "user-id", "12345"),
		},
		{
			name:  "dispatch",
			patch: NewDispatchPatch("h22", "custom-event", `{"detail":"value"}`),
		},
		// NOTE: eval test case removed - PatchEval removed for security
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create patches frame
			pf := &PatchesFrame{
				Seq:     1,
				Patches: []Patch{tc.patch},
			}

			// Encode
			encoded := EncodePatches(pf)
			if len(encoded) == 0 {
				t.Fatal("Encoded patches is empty")
			}

			// Decode
			decoded, err := DecodePatches(encoded)
			if err != nil {
				t.Fatalf("DecodePatches() error = %v", err)
			}

			if decoded.Seq != pf.Seq {
				t.Errorf("Seq = %d, want %d", decoded.Seq, pf.Seq)
			}
			if len(decoded.Patches) != 1 {
				t.Fatalf("Patches count = %d, want 1", len(decoded.Patches))
			}

			got := decoded.Patches[0]
			verifyPatch(t, tc.name, got, tc.patch)
		})
	}
}

func verifyPatch(t *testing.T, _ string, got, want Patch) {
	t.Helper()

	if got.Op != want.Op {
		t.Errorf("Op = %v, want %v", got.Op, want.Op)
	}
	if got.HID != want.HID {
		t.Errorf("HID = %q, want %q", got.HID, want.HID)
	}
	if got.Key != want.Key {
		t.Errorf("Key = %q, want %q", got.Key, want.Key)
	}
	if got.Value != want.Value {
		t.Errorf("Value = %q, want %q", got.Value, want.Value)
	}
	if got.ParentID != want.ParentID {
		t.Errorf("ParentID = %q, want %q", got.ParentID, want.ParentID)
	}
	if got.Index != want.Index {
		t.Errorf("Index = %d, want %d", got.Index, want.Index)
	}
	if got.Bool != want.Bool {
		t.Errorf("Bool = %v, want %v", got.Bool, want.Bool)
	}
	if got.X != want.X {
		t.Errorf("X = %d, want %d", got.X, want.X)
	}
	if got.Y != want.Y {
		t.Errorf("Y = %d, want %d", got.Y, want.Y)
	}
	if got.Behavior != want.Behavior {
		t.Errorf("Behavior = %v, want %v", got.Behavior, want.Behavior)
	}
}

func TestPatchesFrameMultiple(t *testing.T) {
	pf := &PatchesFrame{
		Seq: 42,
		Patches: []Patch{
			NewSetTextPatch("h1", "Updated text"),
			NewSetAttrPatch("h2", "class", "active"),
			NewRemoveNodePatch("h3"),
			NewFocusPatch("h4"),
		},
	}

	encoded := EncodePatches(pf)
	decoded, err := DecodePatches(encoded)
	if err != nil {
		t.Fatalf("DecodePatches() error = %v", err)
	}

	if decoded.Seq != 42 {
		t.Errorf("Seq = %d, want 42", decoded.Seq)
	}
	if len(decoded.Patches) != 4 {
		t.Errorf("Patches count = %d, want 4", len(decoded.Patches))
	}

	// Verify each patch
	if decoded.Patches[0].Op != PatchSetText {
		t.Errorf("Patch 0 Op = %v, want SetText", decoded.Patches[0].Op)
	}
	if decoded.Patches[1].Op != PatchSetAttr {
		t.Errorf("Patch 1 Op = %v, want SetAttr", decoded.Patches[1].Op)
	}
	if decoded.Patches[2].Op != PatchRemoveNode {
		t.Errorf("Patch 2 Op = %v, want RemoveNode", decoded.Patches[2].Op)
	}
	if decoded.Patches[3].Op != PatchFocus {
		t.Errorf("Patch 3 Op = %v, want Focus", decoded.Patches[3].Op)
	}
}

func TestPatchOpString(t *testing.T) {
	tests := []struct {
		op   PatchOp
		want string
	}{
		{PatchSetText, "SetText"},
		{PatchSetAttr, "SetAttr"},
		{PatchRemoveAttr, "RemoveAttr"},
		{PatchInsertNode, "InsertNode"},
		{PatchRemoveNode, "RemoveNode"},
		{PatchMoveNode, "MoveNode"},
		{PatchReplaceNode, "ReplaceNode"},
		{PatchSetValue, "SetValue"},
		{PatchSetChecked, "SetChecked"},
		{PatchSetSelected, "SetSelected"},
		{PatchFocus, "Focus"},
		{PatchBlur, "Blur"},
		{PatchScrollTo, "ScrollTo"},
		{PatchAddClass, "AddClass"},
		{PatchRemoveClass, "RemoveClass"},
		{PatchToggleClass, "ToggleClass"},
		{PatchSetStyle, "SetStyle"},
		{PatchRemoveStyle, "RemoveStyle"},
		{PatchSetData, "SetData"},
		{PatchDispatch, "Dispatch"},
		// NOTE: PatchEval removed for security
		{PatchOp(0xFF), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.op.String(); got != tc.want {
			t.Errorf("PatchOp(%d).String() = %q, want %q", tc.op, got, tc.want)
		}
	}
}

func TestEmptyPatchesFrame(t *testing.T) {
	pf := &PatchesFrame{
		Seq:     1,
		Patches: []Patch{},
	}

	encoded := EncodePatches(pf)
	decoded, err := DecodePatches(encoded)
	if err != nil {
		t.Fatalf("DecodePatches() error = %v", err)
	}

	if len(decoded.Patches) != 0 {
		t.Errorf("Patches count = %d, want 0", len(decoded.Patches))
	}
}

func TestPatchEncodingSize(t *testing.T) {
	// Verify SetText patch is compact (target: <20 bytes for short text)
	p := NewSetTextPatch("h1", "Hello")
	pf := &PatchesFrame{Seq: 1, Patches: []Patch{p}}
	encoded := EncodePatches(pf)
	if len(encoded) > 20 {
		t.Errorf("SetText patch size = %d bytes, want <= 20", len(encoded))
	}
}

func BenchmarkEncodePatch(b *testing.B) {
	pf := &PatchesFrame{
		Seq: 1,
		Patches: []Patch{
			NewSetTextPatch("h1", "Hello, World!"),
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodePatches(pf)
	}
}

func BenchmarkDecodePatch(b *testing.B) {
	pf := &PatchesFrame{
		Seq: 1,
		Patches: []Patch{
			NewSetTextPatch("h1", "Hello, World!"),
		},
	}
	encoded := EncodePatches(pf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePatches(encoded)
	}
}

func BenchmarkEncodePatches100(b *testing.B) {
	patches := make([]Patch, 100)
	for i := range patches {
		patches[i] = NewSetTextPatch("h1", "test value")
	}
	pf := &PatchesFrame{Seq: 1, Patches: patches}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodePatches(pf)
	}
}

func BenchmarkDecodePatches100(b *testing.B) {
	patches := make([]Patch, 100)
	for i := range patches {
		patches[i] = NewSetTextPatch("h1", "test value")
	}
	pf := &PatchesFrame{Seq: 1, Patches: patches}
	encoded := EncodePatches(pf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePatches(encoded)
	}
}
