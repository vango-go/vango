package server

import (
	"context"
	"testing"
)

func TestWithUserAndUserFromContext(t *testing.T) {
	if UserFromContext(nil) != nil {
		t.Fatal("UserFromContext(nil) != nil")
	}

	ctx := context.Background()
	ctx = WithUser(ctx, "u1")
	if got := UserFromContext(ctx); got != "u1" {
		t.Fatalf("UserFromContext()=%v, want %q", got, "u1")
	}
}

