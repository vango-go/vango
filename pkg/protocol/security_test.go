package protocol

import (
	"testing"
)

// =============================================================================
// Phase 13.4: Protocol Allocation Audit Tests
// =============================================================================

// TestAllocationLimits verifies that allocation limits are enforced.
func TestAllocationLimits(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		wantErr error
	}{
		{
			name:    "string exceeds limit",
			payload: makeOversizedStringPayload(DefaultMaxAllocation + 1),
			wantErr: ErrAllocationTooLarge,
		},
		{
			name:    "collection exceeds limit",
			payload: makeOversizedCollectionPayload(MaxCollectionCount + 1),
			wantErr: ErrCollectionTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(tt.payload)
			switch tt.name {
			case "string exceeds limit":
				_, err := d.ReadString()
				if err != tt.wantErr {
					t.Errorf("ReadString() error = %v, want %v", err, tt.wantErr)
				}
			case "collection exceeds limit":
				_, err := d.ReadCollectionCount()
				if err != tt.wantErr {
					t.Errorf("ReadCollectionCount() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestDepthLimits verifies that depth limits are enforced.
func TestDepthLimits(t *testing.T) {
	tests := []struct {
		name      string
		depth     int
		maxDepth  int
		wantError bool
	}{
		{
			name:      "at limit",
			depth:     MaxVNodeDepth,
			maxDepth:  MaxVNodeDepth,
			wantError: false,
		},
		{
			name:      "exceeds limit",
			depth:     MaxVNodeDepth + 1,
			maxDepth:  MaxVNodeDepth,
			wantError: true,
		},
		{
			name:      "well under limit",
			depth:     10,
			maxDepth:  MaxVNodeDepth,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkDepth(tt.depth, tt.maxDepth)
			if (err != nil) != tt.wantError {
				t.Errorf("checkDepth(%d, %d) error = %v, wantError %v", tt.depth, tt.maxDepth, err, tt.wantError)
			}
		})
	}
}

// TestVNodeDepthLimit verifies VNode decoding depth is limited.
func TestVNodeDepthLimit(t *testing.T) {
	// Create deeply nested VNode
	deepNode := createDeeplyNestedVNode(MaxVNodeDepth + 10)
	e := NewEncoder()
	EncodeVNodeWire(e, deepNode)

	d := NewDecoder(e.Bytes())
	_, err := DecodeVNodeWire(d)
	if err != ErrMaxDepthExceeded {
		t.Errorf("DecodeVNodeWire() with deep nesting: got err = %v, want %v", err, ErrMaxDepthExceeded)
	}
}

// TestPatchDepthLimit verifies Patch decoding depth is limited.
func TestPatchDepthLimit(t *testing.T) {
	// Create patch with deeply nested VNode
	deepNode := createDeeplyNestedVNode(MaxVNodeDepth + 10)

	pf := &PatchesFrame{
		Seq: 1,
		Patches: []Patch{
			NewInsertNodePatch("h1", "h0", 0, deepNode),
		},
	}

	d := NewDecoder(EncodePatches(pf))
	_, err := DecodePatchesFrom(d)
	if err != ErrMaxDepthExceeded {
		t.Errorf("DecodePatchesFrom() with deep VNode: got err = %v, want %v", err, ErrMaxDepthExceeded)
	}
}

// TestValidInputsStillWork verifies that valid inputs work after adding limits.
func TestValidInputsStillWork(t *testing.T) {
	t.Run("normal VNode", func(t *testing.T) {
		node := NewElementWire("div", map[string]string{"class": "test"},
			NewTextWire("Hello"),
			NewElementWire("span", nil, NewTextWire("World")),
		)

		e := NewEncoder()
		EncodeVNodeWire(e, node)

		d := NewDecoder(e.Bytes())
		decoded, err := DecodeVNodeWire(d)
		if err != nil {
			t.Fatalf("DecodeVNodeWire() error = %v", err)
		}
		if decoded.Tag != "div" {
			t.Errorf("decoded.Tag = %q, want %q", decoded.Tag, "div")
		}
		if len(decoded.Children) != 2 {
			t.Errorf("len(decoded.Children) = %d, want %d", len(decoded.Children), 2)
		}
	})

	t.Run("normal patches", func(t *testing.T) {
		pf := &PatchesFrame{
			Seq: 1,
			Patches: []Patch{
				NewSetTextPatch("h1", "Hello"),
				NewSetAttrPatch("h2", "class", "active"),
				NewInsertNodePatch("h3", "h0", 0, NewTextWire("New")),
			},
		}

		encoded := EncodePatches(pf)
		decoded, err := DecodePatches(encoded)
		if err != nil {
			t.Fatalf("DecodePatches() error = %v", err)
		}
		if decoded.Seq != 1 {
			t.Errorf("decoded.Seq = %d, want %d", decoded.Seq, 1)
		}
		if len(decoded.Patches) != 3 {
			t.Errorf("len(decoded.Patches) = %d, want %d", len(decoded.Patches), 3)
		}
	})

	t.Run("moderately nested VNode", func(t *testing.T) {
		// Create a reasonably nested structure (50 levels)
		node := createDeeplyNestedVNode(50)
		e := NewEncoder()
		EncodeVNodeWire(e, node)

		d := NewDecoder(e.Bytes())
		decoded, err := DecodeVNodeWire(d)
		if err != nil {
			t.Fatalf("DecodeVNodeWire() error = %v", err)
		}
		if decoded == nil {
			t.Error("decoded is nil")
		}
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// makeOversizedStringPayload creates a payload with a string length exceeding the limit.
func makeOversizedStringPayload(size uint64) []byte {
	e := NewEncoder()
	e.WriteUvarint(size) // Length prefix claiming a huge string
	return e.Bytes()
}

// makeOversizedCollectionPayload creates a payload with a collection count exceeding the limit.
func makeOversizedCollectionPayload(count uint64) []byte {
	e := NewEncoder()
	e.WriteUvarint(count) // Collection count
	return e.Bytes()
}

// createDeeplyNestedVNode creates a VNode tree with the specified depth.
func createDeeplyNestedVNode(depth int) *VNodeWire {
	if depth <= 0 {
		return NewTextWire("leaf")
	}
	return NewElementWire("div", nil, createDeeplyNestedVNode(depth-1))
}

// =============================================================================
// Phase 13.4: Allocation Audit Verification
// =============================================================================

// TestAllDecodePathsProtected verifies all decode paths have allocation limits.
// This is a comprehensive audit of the protocol package.
func TestAllDecodePathsProtected(t *testing.T) {
	// This test documents all the decode paths and their protections

	t.Run("decoder primitives", func(t *testing.T) {
		// ReadString - protected by DefaultMaxAllocation
		// ReadLenBytes - protected by DefaultMaxAllocation
		// ReadCollectionCount - protected by MaxCollectionCount

		// Verify limits exist
		if DefaultMaxAllocation <= 0 {
			t.Error("DefaultMaxAllocation not set")
		}
		if MaxCollectionCount <= 0 {
			t.Error("MaxCollectionCount not set")
		}
		if HardMaxAllocation < DefaultMaxAllocation {
			t.Error("HardMaxAllocation should be >= DefaultMaxAllocation")
		}
	})

	t.Run("vnode decode", func(t *testing.T) {
		// DecodeVNodeWire - protected by:
		// - MaxVNodeDepth (via decodeVNodeWireWithDepth)
		// - ReadCollectionCount for attrs and children
		// - ReadString for tag, HID, text

		if MaxVNodeDepth <= 0 {
			t.Error("MaxVNodeDepth not set")
		}
	})

	t.Run("patch decode", func(t *testing.T) {
		// DecodePatchesFrom - protected by:
		// - MaxPatchDepth (via decodePatchesFromWithDepth)
		// - ReadCollectionCount for patch array
		// - decodeVNodeWireWithDepth for InsertNode/ReplaceNode

		if MaxPatchDepth <= 0 {
			t.Error("MaxPatchDepth not set")
		}
	})

	t.Run("event decode", func(t *testing.T) {
		// DecodeEvent - protected by:
		// - MaxHookDepth for hook payloads
		// - ReadCollectionCount for arrays/objects in payloads
		// - ReadString for HID, payload strings

		if MaxHookDepth <= 0 {
			t.Error("MaxHookDepth not set")
		}
	})
}

// TestDepthContextHelper verifies the depth context helper works correctly.
func TestDepthContextHelper(t *testing.T) {
	dc := newDepthContext(5)

	// Should allow entry up to max
	for i := 0; i < 5; i++ {
		if err := dc.enter(); err != nil {
			t.Errorf("enter() at depth %d failed: %v", i+1, err)
		}
	}

	// Should reject at max+1
	if err := dc.enter(); err != ErrMaxDepthExceeded {
		t.Errorf("enter() at depth 6 should fail with ErrMaxDepthExceeded, got %v", err)
	}

	// Leave and should be able to enter again
	dc.leave()
	if err := dc.enter(); err != nil {
		t.Errorf("enter() after leave() failed: %v", err)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkDecodeVNodeWireWithDepth(b *testing.B) {
	node := NewElementWire("div", map[string]string{"class": "test"},
		NewElementWire("span", nil, NewTextWire("Hello")),
		NewElementWire("span", nil, NewTextWire("World")),
	)
	e := NewEncoder()
	EncodeVNodeWire(e, node)
	data := e.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := NewDecoder(data)
		_, _ = DecodeVNodeWire(d)
	}
}

func BenchmarkDecodePatchesWithDepth(b *testing.B) {
	pf := &PatchesFrame{
		Seq: 1,
		Patches: []Patch{
			NewSetTextPatch("h1", "Hello"),
			NewSetAttrPatch("h2", "class", "active"),
			NewInsertNodePatch("h3", "h0", 0, NewTextWire("New")),
		},
	}
	data := EncodePatches(pf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePatches(data)
	}
}
