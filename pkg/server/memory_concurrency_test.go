package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

type memUsageComponent struct {
	count *vango.Signal[int]
}

func newMemUsageComponent() *memUsageComponent {
	return &memUsageComponent{count: vango.NewSignal(0)}
}

func (c *memUsageComponent) Render() *vdom.VNode {
	return vdom.Div(
		vdom.Button(
			vdom.OnClick(func() {
				c.count.Set(c.count.Get() + 1)
			}),
			vdom.Text("inc"),
		),
		vdom.Div(vdom.Text(fmt.Sprintf("%d", c.count.Get()))),
	)
}

func findClickHID(t *testing.T, sess *Session) string {
	t.Helper()
	sess.stateMu.RLock()
	defer sess.stateMu.RUnlock()
	for key := range sess.handlers {
		if strings.HasSuffix(key, "_onclick") {
			return strings.TrimSuffix(key, "_onclick")
		}
	}
	t.Fatal("onclick handler not found")
	return ""
}

func TestSessionMemoryUsageConcurrent(t *testing.T) {
	cfg := DefaultSessionConfig()
	sess := newSession(nil, "", cfg, slog.Default())
	sess.MountRoot(newMemUsageComponent())

	hid := findClickHID(t, sess)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var seq uint64 = 1
		for {
			select {
			case <-ctx.Done():
				return
			default:
				sess.handleEvent(&Event{HID: hid, Type: protocol.EventClick, Seq: seq})
				seq++
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_ = sess.MemoryUsage()
			}
		}
	}()

	wg.Wait()
}

func TestSessionManagerCheckMemoryPressureConcurrent(t *testing.T) {
	cfg := DefaultSessionConfig()
	limits := DefaultSessionLimits()
	limits.MaxMemoryPerSession = 1 << 30

	sm := NewSessionManager(cfg, limits, slog.Default())
	t.Cleanup(func() { sm.Shutdown() })

	sess := newSession(nil, "", cfg, slog.Default())
	sess.MountRoot(newMemUsageComponent())
	hid := findClickHID(t, sess)

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				sm.CheckMemoryPressure()
			}
		}
	}()

	go func() {
		defer wg.Done()
		var seq uint64 = 1
		for {
			select {
			case <-ctx.Done():
				return
			default:
				sess.handleEvent(&Event{HID: hid, Type: protocol.EventClick, Seq: seq})
				seq++
			}
		}
	}()

	wg.Wait()
}
