package router

import (
	"testing"
)

func TestLink(t *testing.T) {
	node := Link("/users", "Users")

	if node == nil {
		t.Fatal("Link returned nil")
	}
	if node.Tag != "a" {
		t.Errorf("Tag = %q, want %q", node.Tag, "a")
	}

	// Check href
	if node.Props["href"] != "/users" {
		t.Errorf("href = %v, want %q", node.Props["href"], "/users")
	}

	// Check data-vango-link attribute (canonical marker for SPA navigation)
	if _, exists := node.Props["data-vango-link"]; !exists {
		t.Error("data-vango-link attribute should be present")
	}
}

func TestLinkWithPrefetch(t *testing.T) {
	node := LinkWithPrefetch("/articles", "Articles")

	if node == nil {
		t.Fatal("LinkWithPrefetch returned nil")
	}

	// Check data-prefetch
	if _, exists := node.Props["data-prefetch"]; !exists {
		t.Error("data-prefetch attribute should be present")
	}

	// Check data-vango-link
	if _, exists := node.Props["data-vango-link"]; !exists {
		t.Error("data-vango-link attribute should be present")
	}
}

func TestActiveLink(t *testing.T) {
	node := ActiveLink("/dashboard", "nav-active", true, "Dashboard")

	if node == nil {
		t.Fatal("ActiveLink returned nil")
	}

	// Check data-active-class
	if node.Props["data-active-class"] != "nav-active" {
		t.Errorf("data-active-class = %v, want %q", node.Props["data-active-class"], "nav-active")
	}

	// Check data-active-exact
	if node.Props["data-active-exact"] != "true" {
		t.Errorf("data-active-exact = %v, want %q", node.Props["data-active-exact"], "true")
	}
}

func TestActiveLinkPartialMatch(t *testing.T) {
	node := ActiveLink("/blog", "active", false, "Blog")

	if node == nil {
		t.Fatal("ActiveLink returned nil")
	}

	// Should NOT have exact match
	if _, ok := node.Props["data-active-exact"]; ok {
		t.Error("data-active-exact should not be set for partial match")
	}
}

func TestNavLink(t *testing.T) {
	node := NavLink("/settings", "Settings")

	if node == nil {
		t.Fatal("NavLink returned nil")
	}

	// Should use "active" class
	if node.Props["data-active-class"] != "active" {
		t.Errorf("data-active-class = %v, want %q", node.Props["data-active-class"], "active")
	}

	// Should be exact match
	if node.Props["data-active-exact"] != "true" {
		t.Errorf("data-active-exact = %v, want %q", node.Props["data-active-exact"], "true")
	}
}

func TestPrefetch(t *testing.T) {
	attr := Prefetch()

	if attr.Key != "data-prefetch" {
		t.Errorf("Key = %q, want %q", attr.Key, "data-prefetch")
	}
	// Empty string value - presence is what matters
	if attr.Value != "" {
		t.Errorf("Value = %v, want empty string", attr.Value)
	}
}

func TestDataLink(t *testing.T) {
	// DataLink is deprecated but still returns the canonical marker
	attr := DataLink()

	if attr.Key != "data-vango-link" {
		t.Errorf("Key = %q, want %q", attr.Key, "data-vango-link")
	}
	// Empty string value - presence is what matters
	if attr.Value != "" {
		t.Errorf("Value = %v, want empty string", attr.Value)
	}
}

func TestVangoLink(t *testing.T) {
	attr := VangoLink()

	if attr.Key != "data-vango-link" {
		t.Errorf("Key = %q, want %q", attr.Key, "data-vango-link")
	}
	// Empty string value - presence is what matters
	if attr.Value != "" {
		t.Errorf("Value = %v, want empty string", attr.Value)
	}
}
