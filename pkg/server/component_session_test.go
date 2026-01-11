package server

import "testing"

func TestComponentInstance_Session(t *testing.T) {
	s := NewMockSession()
	inst := newComponentInstance(staticComponent{}, nil, s)
	if inst.Session() != s {
		t.Fatal("Session() did not return owning session")
	}
	inst.Dispose()
	if inst.Session() != nil {
		t.Fatal("Session() != nil after Dispose")
	}
}

