/**
 * WebSocket Connection Manager
 *
 * Handles WebSocket connection lifecycle, reconnection, and message routing.
 */

export class WebSocketManager {
    constructor(client, options = {}) {
        this.client = client;
        this.options = {
            reconnect: options.reconnect !== false,
            reconnectInterval: options.reconnectInterval || 1000,
            reconnectMaxInterval: options.reconnectMaxInterval || 30000,
            heartbeatInterval: options.heartbeatInterval || 30000,
            ...options,
        };

        this.ws = null;
        this.connected = false;
        this.sessionId = null;
        this.reconnectAttempts = 0;
        this.heartbeatTimer = null;
        this.messageQueue = [];
    }

    /**
     * Connect to WebSocket server
     */
    connect(url) {
        if (this.ws) {
            this.ws.close();
        }

        this.url = url;
        this.ws = new WebSocket(url);
        this.ws.binaryType = 'arraybuffer';

        this.ws.onopen = () => this._onOpen();
        this.ws.onclose = (e) => this._onClose(e);
        this.ws.onerror = (e) => this._onError(e);
        this.ws.onmessage = (e) => this._onMessage(e);
    }

    /**
     * Handle WebSocket open
     */
    _onOpen() {
        if (this.client.options.debug) {
            console.log('[Vango] WebSocket connected');
        }

        this.reconnectAttempts = 0;

        // Send handshake
        this._sendHandshake();

        // Start heartbeat
        this._startHeartbeat();
    }

    /**
     * Send handshake message (JSON, not binary)
     */
    _sendHandshake() {
        const handshake = {
            type: 'handshake',
            version: '1.0',
            csrf: window.__VANGO_CSRF__ || '',
            session: this.sessionId || '',
            path: location.pathname,
            viewport: {
                width: window.innerWidth,
                height: window.innerHeight,
            },
        };

        this.ws.send(JSON.stringify(handshake));
    }

    /**
     * Handle incoming message
     */
    _onMessage(event) {
        // First message after handshake is JSON acknowledgment
        if (!this.connected) {
            try {
                const ack = JSON.parse(event.data);
                if (ack.type === 'handshake_ack') {
                    this.connected = true;
                    this.sessionId = ack.session;
                    this.client._onConnected();

                    // Send queued messages
                    this._flushQueue();

                    if (this.client.options.debug) {
                        console.log('[Vango] Handshake complete, session:', this.sessionId);
                    }
                    return;
                }
            } catch (e) {
                // Not JSON, treat as binary
            }
        }

        // All other messages are binary
        if (event.data instanceof ArrayBuffer) {
            this.client._handleBinaryMessage(new Uint8Array(event.data));
        }
    }

    /**
     * Handle WebSocket close
     */
    _onClose(event) {
        const wasConnected = this.connected;
        this.connected = false;
        this._stopHeartbeat();

        if (this.client.options.debug) {
            console.log('[Vango] WebSocket closed:', event.code, event.reason);
        }

        if (wasConnected) {
            this.client._onDisconnected();
        }

        // Reconnect if enabled and not a clean close
        if (this.options.reconnect && !event.wasClean) {
            this._scheduleReconnect();
        }
    }

    /**
     * Handle WebSocket error
     */
    _onError(event) {
        if (this.client.options.debug) {
            console.error('[Vango] WebSocket error:', event);
        }
        this.client._onError(new Error('WebSocket error'));
    }

    /**
     * Schedule reconnection with exponential backoff
     */
    _scheduleReconnect() {
        const delay = Math.min(
            this.options.reconnectInterval * Math.pow(2, this.reconnectAttempts),
            this.options.reconnectMaxInterval
        );

        this.reconnectAttempts++;

        if (this.client.options.debug) {
            console.log(`[Vango] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
        }

        setTimeout(() => this.connect(this.url), delay);
    }

    /**
     * Start heartbeat timer
     */
    _startHeartbeat() {
        this.heartbeatTimer = setInterval(() => {
            if (this.connected) {
                this._sendPing();
            }
        }, this.options.heartbeatInterval);
    }

    /**
     * Stop heartbeat timer
     */
    _stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }

    /**
     * Send ping (control message)
     */
    _sendPing() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            // Frame format: [type:1][subtype:1]
            // Control frame (0x02), Ping subtype (0x01)
            const buffer = new Uint8Array([0x02, 0x01]);
            this.ws.send(buffer);
        }
    }

    /**
     * Send binary message
     */
    send(buffer) {
        if (this.connected && this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(buffer);
            return true;
        } else {
            // Queue for later
            this.messageQueue.push(buffer);
            return false;
        }
    }

    /**
     * Flush queued messages
     */
    _flushQueue() {
        while (this.messageQueue.length > 0 && this.connected) {
            const buffer = this.messageQueue.shift();
            this.ws.send(buffer);
        }
    }

    /**
     * Close connection
     */
    close() {
        this.options.reconnect = false;
        this._stopHeartbeat();

        if (this.ws) {
            this.ws.close(1000, 'Client closing');
            this.ws = null;
        }
    }
}
