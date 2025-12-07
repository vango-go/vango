package vdom

import "testing"

func TestHIDGenerator(t *testing.T) {
	gen := NewHIDGenerator()

	t.Run("sequential generation", func(t *testing.T) {
		h1 := gen.Next()
		h2 := gen.Next()
		h3 := gen.Next()

		if h1 != "h1" {
			t.Errorf("First HID = %v, want h1", h1)
		}
		if h2 != "h2" {
			t.Errorf("Second HID = %v, want h2", h2)
		}
		if h3 != "h3" {
			t.Errorf("Third HID = %v, want h3", h3)
		}
	})

	t.Run("current", func(t *testing.T) {
		gen := NewHIDGenerator()
		gen.Next()
		gen.Next()
		if gen.Current() != 2 {
			t.Errorf("Current() = %v, want 2", gen.Current())
		}
	})

	t.Run("reset", func(t *testing.T) {
		gen := NewHIDGenerator()
		gen.Next()
		gen.Next()
		gen.Reset()

		if gen.Current() != 0 {
			t.Errorf("After reset, Current() = %v, want 0", gen.Current())
		}

		h1 := gen.Next()
		if h1 != "h1" {
			t.Errorf("After reset, Next() = %v, want h1", h1)
		}
	})
}

func TestAssignHIDs(t *testing.T) {
	t.Run("interactive elements get HIDs", func(t *testing.T) {
		tree := Div(
			H1(Text("Title")),
			Button(OnClick(func() {}), Text("Click")),
			Input(OnInput(func() {})),
		)

		gen := NewHIDGenerator()
		AssignHIDs(tree, gen)

		// Div has no event handlers
		if tree.HID != "" {
			t.Errorf("Div should not have HID, got %v", tree.HID)
		}

		// H1 has no event handlers
		if tree.Children[0].HID != "" {
			t.Errorf("H1 should not have HID, got %v", tree.Children[0].HID)
		}

		// Button has onclick
		if tree.Children[1].HID != "h1" {
			t.Errorf("Button HID = %v, want h1", tree.Children[1].HID)
		}

		// Input has oninput
		if tree.Children[2].HID != "h2" {
			t.Errorf("Input HID = %v, want h2", tree.Children[2].HID)
		}
	})

	t.Run("nil node", func(t *testing.T) {
		gen := NewHIDGenerator()
		AssignHIDs(nil, gen) // Should not panic
	})

	t.Run("nested interactive elements", func(t *testing.T) {
		tree := Div(
			Form(
				OnSubmit(func() {}),
				Input(OnInput(func() {})),
				Button(OnClick(func() {}), Text("Submit")),
			),
		)

		gen := NewHIDGenerator()
		AssignHIDs(tree, gen)

		form := tree.Children[0]
		if form.HID != "h1" {
			t.Errorf("Form HID = %v, want h1", form.HID)
		}
		if form.Children[0].HID != "h2" {
			t.Errorf("Input HID = %v, want h2", form.Children[0].HID)
		}
		if form.Children[1].HID != "h3" {
			t.Errorf("Button HID = %v, want h3", form.Children[1].HID)
		}
	})
}

func TestAssignAllHIDs(t *testing.T) {
	tree := Div(
		H1(Text("Title")),
		P(Text("Content")),
	)

	gen := NewHIDGenerator()
	AssignAllHIDs(tree, gen)

	if tree.HID != "h1" {
		t.Errorf("Div HID = %v, want h1", tree.HID)
	}
	if tree.Children[0].HID != "h2" {
		t.Errorf("H1 HID = %v, want h2", tree.Children[0].HID)
	}
	if tree.Children[1].HID != "h3" {
		t.Errorf("P HID = %v, want h3", tree.Children[1].HID)
	}
}

func TestCollectHIDs(t *testing.T) {
	tree := Div(
		Button(OnClick(func() {})),
		Input(OnInput(func() {})),
	)

	gen := NewHIDGenerator()
	AssignHIDs(tree, gen)

	hidMap := CollectHIDs(tree)

	if len(hidMap) != 2 {
		t.Errorf("Expected 2 HIDs, got %d", len(hidMap))
	}

	if hidMap["h1"] == nil {
		t.Error("h1 not found in map")
	}
	if hidMap["h2"] == nil {
		t.Error("h2 not found in map")
	}
}

func TestFindByHID(t *testing.T) {
	button := Button(OnClick(func() {}), Text("Click"))
	tree := Div(
		H1(Text("Title")),
		button,
	)

	gen := NewHIDGenerator()
	AssignHIDs(tree, gen)

	t.Run("found", func(t *testing.T) {
		found := FindByHID(tree, "h1")
		if found != button {
			t.Error("FindByHID did not return button")
		}
	})

	t.Run("not found", func(t *testing.T) {
		found := FindByHID(tree, "h999")
		if found != nil {
			t.Error("Expected nil for non-existent HID")
		}
	})

	t.Run("nil tree", func(t *testing.T) {
		found := FindByHID(nil, "h1")
		if found != nil {
			t.Error("Expected nil for nil tree")
		}
	})
}

func TestCountInteractive(t *testing.T) {
	tree := Div(
		H1(Text("Title")),
		Button(OnClick(func() {})),
		Form(
			OnSubmit(func() {}),
			Input(OnInput(func() {})),
			Input(OnChange(func() {})),
		),
	)

	count := CountInteractive(tree)
	if count != 4 {
		t.Errorf("CountInteractive = %d, want 4", count)
	}
}

func TestCountInteractiveNil(t *testing.T) {
	count := CountInteractive(nil)
	if count != 0 {
		t.Errorf("CountInteractive(nil) = %d, want 0", count)
	}
}

func TestClearHIDs(t *testing.T) {
	tree := Div(
		Button(OnClick(func() {})),
		Input(OnInput(func() {})),
	)

	gen := NewHIDGenerator()
	AssignHIDs(tree, gen)

	// Verify HIDs are assigned
	if tree.Children[0].HID == "" {
		t.Fatal("HID should be assigned before clearing")
	}

	ClearHIDs(tree)

	if tree.HID != "" {
		t.Errorf("Div HID should be cleared, got %v", tree.HID)
	}
	if tree.Children[0].HID != "" {
		t.Errorf("Button HID should be cleared, got %v", tree.Children[0].HID)
	}
	if tree.Children[1].HID != "" {
		t.Errorf("Input HID should be cleared, got %v", tree.Children[1].HID)
	}
}

func TestClearHIDsNil(t *testing.T) {
	ClearHIDs(nil) // Should not panic
}

func TestCopyHIDs(t *testing.T) {
	t.Run("same structure", func(t *testing.T) {
		src := Div(
			H1(Text("Title")),
			P(Text("Content")),
		)
		gen := NewHIDGenerator()
		AssignAllHIDs(src, gen)

		dst := Div(
			H1(Text("New Title")),
			P(Text("New Content")),
		)

		ok := CopyHIDs(src, dst)
		if !ok {
			t.Error("CopyHIDs should return true for same structure")
		}

		if dst.HID != "h1" {
			t.Errorf("dst HID = %v, want h1", dst.HID)
		}
		if dst.Children[0].HID != "h2" {
			t.Errorf("dst.Children[0] HID = %v, want h2", dst.Children[0].HID)
		}
	})

	t.Run("different structure", func(t *testing.T) {
		src := Div(H1(), P())
		dst := Div(H1())

		ok := CopyHIDs(src, dst)
		if ok {
			t.Error("CopyHIDs should return false for different structure")
		}
	})

	t.Run("both nil", func(t *testing.T) {
		ok := CopyHIDs(nil, nil)
		if !ok {
			t.Error("CopyHIDs(nil, nil) should return true")
		}
	})

	t.Run("one nil", func(t *testing.T) {
		ok := CopyHIDs(Div(), nil)
		if ok {
			t.Error("CopyHIDs with one nil should return false")
		}
	})
}

func TestHIDGeneratorConcurrency(t *testing.T) {
	gen := NewHIDGenerator()
	done := make(chan bool)

	// Spawn multiple goroutines generating HIDs
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				gen.Next()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have generated 1000 HIDs
	if gen.Current() != 1000 {
		t.Errorf("Current() = %d, want 1000", gen.Current())
	}
}
