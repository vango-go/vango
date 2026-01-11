package server

import (
	"sync"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
)

func TestSession_EventLoop_ProcessesEventDispatchAndRender(t *testing.T) {
	s := NewMockSession()
	s.MountRoot(onClickComponent{})

	// Use a done channel to wait for dispatch execution.
	var mu sync.Mutex
	dispatchCalls := 0

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.EventLoop()
	}()

	// 1) Event processing path.
	rootHID := s.currentTree.HID
	s.QueueEvent(&Event{
		Seq:  1,
		Type: protocol.EventClick,
		HID:  rootHID,
	})

	// 2) Dispatch path.
	s.Dispatch(func() {
		mu.Lock()
		dispatchCalls++
		mu.Unlock()
	})

	// 3) Render signal path.
	select {
	case s.renderCh <- struct{}{}:
	default:
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := dispatchCalls > 0
		mu.Unlock()
		if done && s.eventCount.Load() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	if dispatchCalls == 0 {
		mu.Unlock()
		t.Fatal("expected dispatch to run on event loop")
	}
	mu.Unlock()
	if s.eventCount.Load() < 1 {
		t.Fatalf("eventCount=%d, want >= 1", s.eventCount.Load())
	}

	s.Close()
	wg.Wait()
}

