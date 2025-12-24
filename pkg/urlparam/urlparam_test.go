package urlparam

import (
	"testing"
	"time"
)

// TestURLParamCreation tests creating URL parameters.
func TestURLParamCreation(t *testing.T) {
	t.Run("SimpleString", func(t *testing.T) {
		param := Param("q", "")
		if param.Get() != "" {
			t.Errorf("Default value: got %v, want empty", param.Get())
		}
	})

	t.Run("WithDefault", func(t *testing.T) {
		param := Param("page", 1)
		if param.Get() != 1 {
			t.Errorf("Default value: got %v, want 1", param.Get())
		}
	})

	t.Run("WithPushMode", func(t *testing.T) {
		param := Param("q", "", Push)
		// Just verify it doesn't panic
		if param == nil {
			t.Error("Param should not be nil")
		}
	})

	t.Run("WithReplaceMode", func(t *testing.T) {
		param := Param("q", "", Replace)
		if param == nil {
			t.Error("Param should not be nil")
		}
	})

	t.Run("WithDebounce", func(t *testing.T) {
		param := Param("q", "", Debounce(300*time.Millisecond))
		if param == nil {
			t.Error("Param should not be nil")
		}
	})

	t.Run("WithEncoding", func(t *testing.T) {
		param := Param("filter", struct{}{}, WithEncoding(EncodingJSON))
		if param == nil {
			t.Error("Param should not be nil")
		}
	})

	t.Run("CombinedOptions", func(t *testing.T) {
		param := Param("q", "", Replace, Debounce(300*time.Millisecond))
		if param == nil {
			t.Error("Param should not be nil")
		}
	})
}

// TestURLParamGetSet tests getting and setting values.
func TestURLParamGetSet(t *testing.T) {
	param := Param("q", "default")

	// Get initial value
	if param.Get() != "default" {
		t.Errorf("Initial Get: got %v, want default", param.Get())
	}

	// Set new value
	param.Set("search term")
	if param.Get() != "search term" {
		t.Errorf("After Set: got %v, want 'search term'", param.Get())
	}

	// Peek
	if param.Peek() != "search term" {
		t.Errorf("Peek: got %v, want 'search term'", param.Peek())
	}
}

// TestURLParamReset tests resetting to default.
func TestURLParamReset(t *testing.T) {
	param := Param("q", "default")

	param.Set("modified")
	if param.Get() != "modified" {
		t.Fatal("Set failed")
	}

	param.Reset()
	if param.Get() != "default" {
		t.Errorf("After Reset: got %v, want default", param.Get())
	}
}

// TestURLParamUpdate tests the Update function.
func TestURLParamUpdate(t *testing.T) {
	param := Param("count", 0)

	param.Update(func(v int) int { return v + 1 })
	if param.Get() != 1 {
		t.Errorf("After Update: got %v, want 1", param.Get())
	}

	param.Update(func(v int) int { return v * 10 })
	if param.Get() != 10 {
		t.Errorf("After second Update: got %v, want 10", param.Get())
	}
}

// TestURLParamSetFromURL tests setting from URL parameters.
func TestURLParamSetFromURL(t *testing.T) {
	t.Run("SimpleString", func(t *testing.T) {
		param := Param("q", "")
		params := map[string]string{"q": "search term"}

		err := param.SetFromURL(params)
		if err != nil {
			t.Fatalf("SetFromURL failed: %v", err)
		}
		if param.Get() != "search term" {
			t.Errorf("After SetFromURL: got %v, want 'search term'", param.Get())
		}
	})

	t.Run("IntValue", func(t *testing.T) {
		param := Param("page", 1)
		params := map[string]string{"page": "5"}

		err := param.SetFromURL(params)
		if err != nil {
			t.Fatalf("SetFromURL failed: %v", err)
		}
		if param.Get() != 5 {
			t.Errorf("After SetFromURL: got %v, want 5", param.Get())
		}
	})

	t.Run("MissingKey", func(t *testing.T) {
		param := Param("q", "default")
		params := map[string]string{"other": "value"}

		err := param.SetFromURL(params)
		if err != nil {
			t.Fatalf("SetFromURL failed: %v", err)
		}
		// Should keep default when key is missing
	})
}

// TestURLParamFlatEncoding tests flat struct encoding.
func TestURLParamFlatEncoding(t *testing.T) {
	type Filter struct {
		Category string `url:"cat"`
		SortBy   string `url:"sort"`
		Page     int    `url:"page"`
	}

	param := Param("", Filter{}, WithEncoding(EncodingFlat))

	// Set from URL
	params := map[string]string{
		"cat":  "electronics",
		"sort": "price",
		"page": "2",
	}

	err := param.SetFromURL(params)
	if err != nil {
		t.Fatalf("SetFromURL failed: %v", err)
	}

	got := param.Get()
	if got.Category != "electronics" {
		t.Errorf("Category: got %v, want electronics", got.Category)
	}
	if got.SortBy != "price" {
		t.Errorf("SortBy: got %v, want price", got.SortBy)
	}
	if got.Page != 2 {
		t.Errorf("Page: got %v, want 2", got.Page)
	}
}

// TestURLParamCommaEncoding tests comma-separated encoding.
func TestURLParamCommaEncoding(t *testing.T) {
	param := Param("tags", []string{}, WithEncoding(EncodingComma))

	// Set from URL
	params := map[string]string{"tags": "go,web,api"}

	err := param.SetFromURL(params)
	if err != nil {
		t.Fatalf("SetFromURL failed: %v", err)
	}

	got := param.Get()
	if len(got) != 3 {
		t.Fatalf("Length: got %d, want 3", len(got))
	}
	if got[0] != "go" {
		t.Errorf("got[0]: got %v, want go", got[0])
	}
	if got[1] != "web" {
		t.Errorf("got[1]: got %v, want web", got[1])
	}
	if got[2] != "api" {
		t.Errorf("got[2]: got %v, want api", got[2])
	}
}

// TestURLParamJSONEncoding tests JSON encoding.
func TestURLParamJSONEncoding(t *testing.T) {
	type Filter struct {
		Category string `json:"cat"`
		MaxPrice int    `json:"max_price"`
	}

	param := Param("filter", Filter{}, WithEncoding(EncodingJSON))

	// Set from URL (base64 encoded JSON)
	// {"cat":"electronics","max_price":1000}
	// eyJjYXQiOiJlbGVjdHJvbmljcyIsIm1heF9wcmljZSI6MTAwMH0
	params := map[string]string{"filter": "eyJjYXQiOiJlbGVjdHJvbmljcyIsIm1heF9wcmljZSI6MTAwMH0"}

	err := param.SetFromURL(params)
	if err != nil {
		t.Fatalf("SetFromURL failed: %v", err)
	}

	got := param.Get()
	if got.Category != "electronics" {
		t.Errorf("Category: got %v, want electronics", got.Category)
	}
	if got.MaxPrice != 1000 {
		t.Errorf("MaxPrice: got %v, want 1000", got.MaxPrice)
	}
}

// TestModeConstants tests that mode constants are distinct.
func TestModeConstants(t *testing.T) {
	if ModePush == ModeReplace {
		t.Error("ModePush and ModeReplace should be different")
	}
}

// TestEncodingConstants tests that encoding constants are distinct.
func TestEncodingConstants(t *testing.T) {
	encodings := []Encoding{EncodingFlat, EncodingJSON, EncodingComma}
	seen := make(map[Encoding]bool)

	for _, e := range encodings {
		if seen[e] {
			t.Errorf("Duplicate Encoding value: %v", e)
		}
		seen[e] = true
	}
}
