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

	// Check data-link attribute
	if node.Props["data-link"] != "true" {
		t.Errorf("data-link = %v, want %q", node.Props["data-link"], "true")
	}
}

func TestLinkWithPrefetch(t *testing.T) {
	node := LinkWithPrefetch("/articles", "Articles")

	if node == nil {
		t.Fatal("LinkWithPrefetch returned nil")
	}

	// Check data-prefetch
	if node.Props["data-prefetch"] != "true" {
		t.Errorf("data-prefetch = %v, want %q", node.Props["data-prefetch"], "true")
	}

	// Check data-link
	if node.Props["data-link"] != "true" {
		t.Errorf("data-link = %v, want %q", node.Props["data-link"], "true")
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
	if attr.Value != "true" {
		t.Errorf("Value = %v, want %q", attr.Value, "true")
	}
}

func TestDataLink(t *testing.T) {
	attr := DataLink()

	if attr.Key != "data-link" {
		t.Errorf("Key = %q, want %q", attr.Key, "data-link")
	}
	if attr.Value != "true" {
		t.Errorf("Value = %v, want %q", attr.Value, "true")
	}
}
