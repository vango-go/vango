package server

import (
	"testing"
	"time"
)

func getSessionEventually(t *testing.T, mgr *SessionManager, sessionID string) *Session {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		sess := mgr.Get(sessionID)
		if sess != nil {
			return sess
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for session %q to be registered", sessionID)
	return nil
}

func waitForEventLoopStarted(t *testing.T, sess *Session) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sess.eventLoopRunning.Load() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for session event loop to start")
}

func waitForDetached(t *testing.T, sess *Session) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sess.detached.Load() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for session to detach")
}
