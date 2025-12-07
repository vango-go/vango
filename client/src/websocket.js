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
        this.handshakeComplete = false;
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
     * Send binary ClientHello handshake
     */
    _sendHandshake() {
        const helloBuffer = this.client.codec.encodeClientHello({
            csrf: window.__VANGO_CSRF__ || '',
            sessionId: this.sessionId || '',
            viewportW: window.innerWidth,
            viewportH: window.innerHeight,
        });

        this.ws.send(helloBuffer);

        if (this.client.options.debug) {
            console.log('[Vango] Sent binary ClientHello');
        }
    }

    /**
     * Handle incoming message
     */
    _onMessage(event) {
        // All messages are binary
        if (!(event.data instanceof ArrayBuffer)) {
            if (this.client.options.debug) {
                console.warn('[Vango] Received non-binary message:', event.data);
            }
            return;
        }

        const buffer = new Uint8Array(event.data);

        // First message after connection is ServerHello
        if (!this.handshakeComplete) {
            const hello = this.client.codec.decodeServerHello(buffer);

            if (hello.error) {
                if (this.client.options.debug) {
                    console.error('[Vango] Handshake error:', hello.error);
                }
                this.client._onError(new Error(hello.error));
                return;
            }

            if (!hello.ok) {
                const errorMessages = {
                    0x01: 'Version mismatch',
                    0x02: 'Invalid CSRF token',
                    0x03: 'Session expired',
                    0x04: 'Server busy',
                    0x05: 'Upgrade required',
                    0x06: 'Invalid format',
                    0x07: 'Not authorized',
                    0x08: 'Internal error',
                };
                const msg = errorMessages[hello.status] || `Handshake failed: ${hello.status}`;
                this.client._onError(new Error(msg));
                this.ws.close();
                return;
            }

            this.handshakeComplete = true;
            this.connected = true;
            this.sessionId = hello.sessionId;
            this.client._onConnected();

            // Send queued messages
            this._flushQueue();

            if (this.client.options.debug) {
                console.log('[Vango] Handshake complete, session:', this.sessionId);
            }
            return;
        }

        // All other messages handled by client
        this.client._handleBinaryMessage(buffer);
    }

    /**
     * Handle WebSocket close
     */
    _onClose(event) {
        const wasConnected = this.connected;
        this.connected = false;
        this.handshakeComplete = false;
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
            // Control payload: [control_type:1][timestamp:8]
            // Timestamp as uint64 little-endian
            const timestamp = Date.now();
            const payload = new Uint8Array(9);
            payload[0] = 0x01; // ControlPing
            // Write timestamp as uint64 little-endian
            let ts = timestamp;
            for (let i = 0; i < 8; i++) {
                payload[1 + i] = ts & 0xFF;
                ts = Math.floor(ts / 256);
            }

            // Wrap in frame: [type:1][flags:1][length:2 big-endian][payload]
            const frame = new Uint8Array(4 + payload.length);
            frame[0] = 0x03; // FrameControl
            frame[1] = 0x00; // flags
            frame[2] = (payload.length >> 8) & 0xFF;
            frame[3] = payload.length & 0xFF;
            frame.set(payload, 4);

            this.ws.send(frame);
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
