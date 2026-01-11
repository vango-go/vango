package server

import (
	"testing"

	"github.com/vango-go/vango/pkg/vango"
)

func TestSession_processPendingNavigation_ProcessesCtxNavigateDuringFlush(t *testing.T) {
	s := NewMockSession()

	renderCtx := s.createRenderContext()
	c := renderCtx.(*ctx)

	vango.WithCtx(renderCtx, func() {
		// Set pending nav, then run flush which should process it first.
		c.Navigate("/p")
		s.flush()
	})
}
