/**
 * Vango Thin Client
 *
 * Minimal runtime for server-driven web applications.
 * Connects to server via WebSocket, captures events, applies patches.
 *
 * Target size: < 15KB gzipped
 */

import { BinaryCodec, EventType } from './codec.js';
import { WebSocketManager } from './websocket.js';
import { EventCapture } from './events.js';
import { PatchApplier } from './patches.js';
import { OptimisticUpdates } from './optimistic.js';
import { HookManager } from './hooks/manager.js';
import { ensurePortalRoot } from './hooks/portal.js';
import { ConnectionManager, injectDefaultStyles } from './connection.js';
import { URLManager } from './url.js';
import { PrefManager, MergeStrategy } from './prefs.js';

/**
 * Frame type constants for wire protocol
 * Must match pkg/protocol/frame.go
 */
const FrameType = {
    HANDSHAKE: 0x00,
    EVENT: 0x01,
    PATCHES: 0x02,
    CONTROL: 0x03,
    ACK: 0x04,
    ERROR: 0x05,
};

/**
 * Control message subtypes
 * Must match vango/pkg/protocol/control.go
 */
const ControlType = {
    PING: 0x01,
    PONG: 0x02,
    RESYNC_REQUEST: 0x10,  // Client -> Server: request missed patches
    RESYNC_PATCHES: 0x11,  // Server -> Client: replay missed patches (not used with frame replay)
    RESYNC_FULL: 0x12,     // Server -> Client: full HTML replacement
    CLOSE: 0x20,
};

/**
 * VangoClient - Main client class
 */
export class VangoClient {
    constructor(options = {}) {
        this.options = {
            wsUrl: options.wsUrl || this._defaultWsUrl(),
            reconnect: options.reconnect !== false,
            reconnectInterval: options.reconnectInterval || 1000,
            reconnectMaxInterval: options.reconnectMaxInterval || 30000,
            heartbeatInterval: options.heartbeatInterval || 30000,
            debug: options.debug || false,
            ...options,
        };

        // Core components
        this.codec = new BinaryCodec();
        this.nodeMap = new Map(); // hid -> DOM node
        this.connected = false;
        this.seq = 0; // Event sequence number

        // Patch sequence tracking (Phase 2)
        this.patchSeq = 0;              // Last successfully applied patch sequence
        this.expectedPatchSeq = 1;      // Next expected patch sequence
        this.pendingResync = false;     // Debounce resync requests

        // Sub-systems
        this.wsManager = new WebSocketManager(this, this.options);
        this.patchApplier = new PatchApplier(this);
        this.eventCapture = new EventCapture(this);
        this.optimistic = new OptimisticUpdates(this);
        this.hooks = new HookManager(this);
        this.connection = new ConnectionManager({
            toastOnReconnect: options.toastOnReconnect || window.__VANGO_TOAST_ON_RECONNECT__,
            toastMessage: options.toastMessage || 'Connection restored',
            maxRetries: options.maxRetries || 10,
            baseDelay: options.reconnectInterval || 1000,
            maxDelay: options.reconnectMaxInterval || 30000,
            debug: options.debug,
        });
        this.urlManager = new URLManager(this, { debug: options.debug });
        this.prefs = new PrefManager(this, { debug: options.debug });

        // Callbacks
        this.onConnect = options.onConnect || (() => { });
        this.onDisconnect = options.onDisconnect || (() => { });
        this.onError = options.onError || ((err) => console.error('[Vango]', err));

        // Initialize portal root early to avoid race conditions
        ensurePortalRoot();

        // Inject default connection styles
        injectDefaultStyles();

        // Initialize
        this._buildNodeMap();
        this.wsManager.connect(this.options.wsUrl);
        this.eventCapture.attach();
        this.hooks.initializeFromDOM();
    }

    /**
     * Build initial map of data-hid -> DOM node
     */
    _buildNodeMap() {
        document.querySelectorAll('[data-hid]').forEach(el => {
            const hid = el.dataset.hid;
            this.nodeMap.set(hid, el);
        });

        if (this.options.debug) {
            console.log(`[Vango] Mapped ${this.nodeMap.size} nodes`);
        }
    }

    /**
     * Get default WebSocket URL based on current page
     */
    _defaultWsUrl() {
        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${protocol}//${location.host}/_vango/live`;
    }

    /**
     * Called when WebSocket connection established
     */
    _onConnected() {
        this.connected = true;
        this.connection.onConnect();
        this.onConnect();
    }

    /**
     * Called when WebSocket disconnected
     */
    _onDisconnected() {
        this.connected = false;
        this.connection.onDisconnect();
        this.onDisconnect();
    }

    /**
     * Called on WebSocket error
     */
    _onError(err) {
        this.onError(err);
    }

    /**
     * Handle binary message from server
     */
    _handleBinaryMessage(buffer) {
        if (buffer.length < 4) return; // Need at least frame header

        // Frame header: [type:1][flags:1][length:2 big-endian]
        const frameType = buffer[0];
        const length = (buffer[2] << 8) | buffer[3];
        const payload = buffer.slice(4, 4 + length);

        switch (frameType) {
            case FrameType.PATCHES:
                this._handlePatches(payload);
                break;
            case FrameType.CONTROL:
                this._handleControl(payload);
                break;
            case FrameType.ERROR:
                this._handleServerError(payload);
                break;
            default:
                if (this.options.debug) {
                    console.warn('[Vango] Unknown frame type:', frameType);
                }
        }
    }

    /**
     * Handle patches frame
     */
    _handlePatches(buffer) {
        if (this.options.debug) {
            console.log('[Vango] Received patches buffer, length:', buffer.length);
        }

        const { seq, patches } = this.codec.decodePatches(buffer);

        if (this.options.debug) {
            console.log('[Vango] Decoded', patches.length, 'patches, seq:', seq);
        }

        // Handle duplicate/replayed frame (seq < expected)
        if (seq < this.expectedPatchSeq) {
            if (this.options.debug) {
                console.log('[Vango] Ignoring duplicate patch seq:', seq, 'expected:', this.expectedPatchSeq);
            }
            return;
        }

        // Check for sequence gap (seq > expected means we missed some)
        if (seq > this.expectedPatchSeq) {
            console.warn('[Vango] Patch sequence gap detected',
                'expected:', this.expectedPatchSeq,
                'received:', seq);
            this._requestResync(this.patchSeq);
            return; // Don't apply out-of-order patches
        }

        if (this.options.debug) {
            for (const p of patches) {
                console.log('[Vango] Patch:', p);
            }
            console.log('[Vango] Applying', patches.length, 'patches (seq:', seq, ')');
        }

        // Clear any pending optimistic updates that server confirmed
        this.optimistic.clearPending();

        // Apply patches to DOM
        this.patchApplier.apply(patches);

        // Re-initialize hooks on new elements
        this.hooks.updateFromDOM();

        // Update sequence tracking
        this.patchSeq = seq;
        this.expectedPatchSeq = seq + 1;
        this.pendingResync = false; // Clear resync flag on successful apply

        // Send ACK to server
        this._sendAck(seq);
    }

    /**
     * Handle control message
     */
    _handleControl(buffer) {
        if (buffer.length === 0) return;

        const controlType = buffer[0];

        switch (controlType) {
            case ControlType.PONG:
                // Heartbeat acknowledged
                if (this.options.debug) {
                    console.log('[Vango] Pong received');
                }
                break;
            case ControlType.RESYNC_FULL:
                // Server sends full HTML to replace body
                this._handleResyncFull(buffer.slice(1));
                break;
            case ControlType.CLOSE:
                // Server requesting close
                this.wsManager.close();
                break;
            default:
                if (this.options.debug) {
                    console.log('[Vango] Unknown control type:', controlType);
                }
        }
    }

    /**
     * Handle ResyncFull - replace body content with server-sent HTML
     * Used during session resume to ensure client DOM matches server state
     */
    _handleResyncFull(buffer) {
        if (buffer.length === 0) return;

        // Decode HTML string (varint length-prefixed)
        const { value: html } = this.codec.decodeString(buffer, 0);

        if (this.options.debug) {
            console.log('[Vango] ResyncFull received, replacing body content');
        }

        // Create template to parse HTML
        const template = document.createElement('template');
        template.innerHTML = html;

        // Clear existing node map
        this.nodeMap.clear();

        // Destroy existing hooks before replacing DOM
        this.hooks.destroyAll();

        // Replace body children
        document.body.innerHTML = '';
        while (template.content.firstChild) {
            document.body.appendChild(template.content.firstChild);
        }

        // Rebuild node map from new DOM
        this._buildNodeMap();

        // Reinitialize hooks on new elements
        this.hooks.initializeFromDOM();

        // Reset patch sequence tracking after full resync
        this.patchSeq = 0;
        this.expectedPatchSeq = 1;
        this.pendingResync = false;
    }

    /**
     * Send ACK for received patches
     * Format: [lastSeq:varint][window:varint]
     */
    _sendAck(lastSeq) {
        if (!this.connected) return;

        const payload = this.codec.encodeAck(lastSeq, 100); // window = 100
        const frame = this._encodeFrame(FrameType.ACK, payload);
        this.wsManager.send(frame);

        if (this.options.debug) {
            console.log('[Vango] Sent ACK for seq:', lastSeq);
        }
    }

    /**
     * Request resync for missed patches (with debouncing)
     * Format: [controlType:1][lastSeq:varint]
     */
    _requestResync(lastSeq) {
        // Debounce: only one resync request at a time
        if (this.pendingResync) {
            if (this.options.debug) {
                console.log('[Vango] Resync already pending, skipping');
            }
            return;
        }
        this.pendingResync = true;

        if (this.options.debug) {
            console.log('[Vango] Requesting resync from seq:', lastSeq);
        }

        const payload = this.codec.encodeResyncRequest(lastSeq);
        const frame = this._encodeFrame(FrameType.CONTROL, payload);
        this.wsManager.send(frame);
    }

    /**
     * Handle server error
     *
     * Per spec Section 9.6.2 (error wire format):
     * Wire format: [uint16:code][varint-string:message][bool:fatal]
     * See vango/pkg/protocol/error.go for encoding.
     */
    _handleServerError(buffer) {
        if (buffer.length < 3) return;

        // Decode code (2 bytes, big-endian)
        const code = (buffer[0] << 8) | buffer[1];

        // Decode varint-length string starting at offset 2
        const { value: message, bytesRead } = this.codec.decodeString(buffer, 2);

        // Fatal flag is 1 byte after the message
        const fatalOffset = 2 + bytesRead;
        const fatal = fatalOffset < buffer.length ? buffer[fatalOffset] === 1 : false;

        // Error code registry - matches vango/pkg/protocol/error.go
        const errorMessages = {
            0x0000: 'Unknown error',
            0x0001: 'Invalid frame',
            0x0002: 'Invalid event',
            0x0003: 'Handler not found',
            0x0004: 'Handler panic',
            0x0005: 'Session expired',
            0x0006: 'Rate limited',
            0x0100: 'Server error',
            0x0101: 'Not authorized',
            0x0102: 'Not found',
            0x0103: 'Validation failed',
            0x0104: 'Route error',
        };

        const errorMessage = errorMessages[code] || message || `Unknown error: ${code}`;
        const error = new Error(errorMessage);
        error.code = code;
        error.fatal = fatal;

        this.onError(error);

        // Revert optimistic updates on error
        this.optimistic.revertAll();

        if (fatal) {
            this.wsManager.close();
        }
    }

    /**
     * Send event to server
     */
    sendEvent(type, hid, data = null) {
        if (!this.connected) {
            if (this.options.debug) {
                console.log('[Vango] Not connected, queueing event');
            }
        }

        this.seq++;
        const eventBuffer = this.codec.encodeEvent(this.seq, type, hid, data);

        // Wrap in frame with 4-byte header: [type][flags][length-hi][length-lo]
        const frame = this._encodeFrame(FrameType.EVENT, eventBuffer);

        this.wsManager.send(frame);

        if (this.options.debug) {
            console.log('[Vango] Sent event:', { type, hid, data, seq: this.seq });
        }
    }

    /**
     * Encode a frame with proper header
     * Format: [type:1][flags:1][length:2 big-endian][payload]
     */
    _encodeFrame(frameType, payload) {
        const length = payload.length;
        const frame = new Uint8Array(4 + length);
        frame[0] = frameType;
        frame[1] = 0; // flags
        frame[2] = (length >> 8) & 0xFF; // length high byte
        frame[3] = length & 0xFF; // length low byte
        frame.set(payload, 4);
        return frame;
    }

    /**
     * Send hook event to server
     */
    sendHookEvent(hid, eventName, data = {}) {
        this.sendEvent(EventType.HOOK, hid, { name: eventName, data });
    }

    /**
     * Get DOM node by hydration ID
     */
    getNode(hid) {
        return this.nodeMap.get(hid);
    }

    /**
     * Register a node in the map
     */
    registerNode(hid, node) {
        this.nodeMap.set(hid, node);
    }

    /**
     * Unregister a node from the map
     */
    unregisterNode(hid) {
        this.nodeMap.delete(hid);
    }

    /**
     * Register a custom hook
     */
    registerHook(name, hookClass) {
        this.hooks.register(name, hookClass);
    }

    /**
     * Register a preference
     * @param {string} key - Unique preference key
     * @param {*} defaultValue - Default value
     * @param {Object} options - Configuration options
     * @returns {Pref} Preference instance
     */
    registerPref(key, defaultValue, options = {}) {
        return this.prefs.register(key, defaultValue, options);
    }

    /**
     * Disconnect and cleanup
     */
    destroy() {
        this.eventCapture.detach();
        this.hooks.destroyAll();
        this.prefs.destroy();
        this.wsManager.close();
    }
}

/**
 * Auto-initialize on DOM ready
 */
function init() {
    // Check if already initialized
    if (window.__vango__) {
        return;
    }

    // Read options from script tag data attributes
    const script = document.currentScript || document.querySelector('script[data-vango]');
    const options = {};

    if (script) {
        if (script.dataset.wsUrl) options.wsUrl = script.dataset.wsUrl;
        if (script.dataset.debug) options.debug = script.dataset.debug === 'true';
    }

    // Create global instance
    window.__vango__ = new VangoClient(options);
}

// Initialize when DOM is ready
if (typeof document !== 'undefined') {
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
}

// Export for manual initialization and for the IIFE wrapper
export { EventType } from './codec.js';
export { ConnectionState, ConnectionManager } from './connection.js';
export default VangoClient;
