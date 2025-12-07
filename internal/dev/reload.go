package dev

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// ReloadMessageType represents the type of reload message.
type ReloadMessageType string

const (
	ReloadTypeFull  ReloadMessageType = "reload"
	ReloadTypeCSS   ReloadMessageType = "css"
	ReloadTypeError ReloadMessageType = "error"
	ReloadTypeClear ReloadMessageType = "clear"
)

// ReloadMessage is sent to browsers via WebSocket.
type ReloadMessage struct {
	Type  ReloadMessageType `json:"type"`
	Error string            `json:"error,omitempty"`
	File  string            `json:"file,omitempty"`
}

// ReloadServer manages WebSocket connections for hot reload.
type ReloadServer struct {
	clients  map[*websocket.Conn]bool
	mu       sync.RWMutex
	upgrader websocket.Upgrader
}

// NewReloadServer creates a new reload server.
func NewReloadServer() *ReloadServer {
	return &ReloadServer{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in dev
			},
		},
	}
}

// HandleWebSocket handles WebSocket upgrade and connection.
func (r *ReloadServer) HandleWebSocket(w http.ResponseWriter, req *http.Request) {
	conn, err := r.upgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}

	r.mu.Lock()
	r.clients[conn] = true
	r.mu.Unlock()

	// Keep connection alive until client disconnects
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	r.mu.Lock()
	delete(r.clients, conn)
	r.mu.Unlock()
	conn.Close()
}

// NotifyReload sends a full page reload message to all clients.
func (r *ReloadServer) NotifyReload() {
	r.broadcast(ReloadMessage{Type: ReloadTypeFull})
}

// NotifyCSS sends a CSS-only reload message to all clients.
func (r *ReloadServer) NotifyCSS(file string) {
	r.broadcast(ReloadMessage{Type: ReloadTypeCSS, File: file})
}

// NotifyError sends an error message to all clients.
func (r *ReloadServer) NotifyError(errMsg string) {
	r.broadcast(ReloadMessage{Type: ReloadTypeError, Error: errMsg})
}

// ClearError clears the error overlay on all clients.
func (r *ReloadServer) ClearError() {
	r.broadcast(ReloadMessage{Type: ReloadTypeClear})
}

// broadcast sends a message to all connected clients.
func (r *ReloadServer) broadcast(msg ReloadMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	r.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	r.mu.RUnlock()

	for _, client := range clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			r.mu.Lock()
			delete(r.clients, client)
			r.mu.Unlock()
			client.Close()
		}
	}
}

// ClientCount returns the number of connected clients.
func (r *ReloadServer) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Close closes all client connections.
func (r *ReloadServer) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for client := range r.clients {
		client.Close()
		delete(r.clients, client)
	}
}

// DevClientScript returns the JavaScript for hot reload.
// This is injected into the page in development mode.
const DevClientScript = `
<script>
(function() {
    'use strict';

    var reconnectDelay = 1000;
    var maxReconnectDelay = 30000;
    var ws = null;

    function connect() {
        var protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(protocol + '//' + location.host + '/_vango/reload');

        ws.onopen = function() {
            console.log('[Vango] Hot reload connected');
            reconnectDelay = 1000;
            clearErrorOverlay();
        };

        ws.onmessage = function(e) {
            var msg;
            try {
                msg = JSON.parse(e.data);
            } catch (err) {
                return;
            }

            switch (msg.type) {
                case 'reload':
                    console.log('[Vango] Reloading...');
                    location.reload();
                    break;

                case 'css':
                    console.log('[Vango] Reloading CSS...');
                    reloadCSS();
                    break;

                case 'error':
                    console.error('[Vango] Build error:', msg.error);
                    showErrorOverlay(msg.error);
                    break;

                case 'clear':
                    clearErrorOverlay();
                    break;
            }
        };

        ws.onclose = function() {
            console.log('[Vango] Connection lost, reconnecting in', reconnectDelay + 'ms');
            setTimeout(function() {
                reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
                connect();
            }, reconnectDelay);
        };

        ws.onerror = function() {
            ws.close();
        };
    }

    function reloadCSS() {
        var links = document.querySelectorAll('link[rel="stylesheet"]');
        links.forEach(function(link) {
            var href = link.href;
            var url = new URL(href);
            url.searchParams.set('_reload', Date.now());
            link.href = url.toString();
        });
    }

    function showErrorOverlay(error) {
        clearErrorOverlay();

        var overlay = document.createElement('div');
        overlay.id = 'vango-error-overlay';
        overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.9);color:#fff;font-family:monospace;font-size:14px;padding:20px;overflow:auto;z-index:999999;';

        var content = document.createElement('div');
        content.style.cssText = 'max-width:800px;margin:0 auto;';

        var title = document.createElement('h2');
        title.style.cssText = 'color:#ff5555;margin:0 0 20px;';
        title.textContent = 'Build Error';

        var pre = document.createElement('pre');
        pre.style.cssText = 'white-space:pre-wrap;word-wrap:break-word;background:#1a1a1a;padding:20px;border-radius:8px;border:1px solid #333;';
        pre.textContent = error;

        var hint = document.createElement('p');
        hint.style.cssText = 'margin-top:20px;color:#888;';
        hint.textContent = 'Fix the error and save to reload.';

        content.appendChild(title);
        content.appendChild(pre);
        content.appendChild(hint);
        overlay.appendChild(content);
        document.body.appendChild(overlay);
    }

    function clearErrorOverlay() {
        var overlay = document.getElementById('vango-error-overlay');
        if (overlay) {
            overlay.remove();
        }
    }

    // Connect on load
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', connect);
    } else {
        connect();
    }
})();
</script>
`
