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

/**
 * Frame type constants for wire protocol
 */
const FrameType = {
    EVENT: 0x00,
    PATCHES: 0x01,
    CONTROL: 0x02,
    ERROR: 0x03,
};

/**
 * Control message subtypes
 */
const ControlType = {
    PING: 0x01,
    PONG: 0x02,
    RESYNC: 0x03,
    CLOSE: 0x04,
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
        this.seq = 0;

        // Sub-systems
        this.wsManager = new WebSocketManager(this, this.options);
        this.patchApplier = new PatchApplier(this);
        this.eventCapture = new EventCapture(this);
        this.optimistic = new OptimisticUpdates(this);
        this.hooks = new HookManager(this);

        // Callbacks
        this.onConnect = options.onConnect || (() => { });
        this.onDisconnect = options.onDisconnect || (() => { });
        this.onError = options.onError || ((err) => console.error('[Vango]', err));

        // Initialize portal root early to avoid race conditions
        ensurePortalRoot();

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
        this.onConnect();
    }

    /**
     * Called when WebSocket disconnected
     */
    _onDisconnected() {
        this.connected = false;
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
        if (buffer.length === 0) return;

        const frameType = buffer[0];
        const payload = buffer.slice(1);

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
        const { seq, patches } = this.codec.decodePatches(buffer);

        if (this.options.debug) {
            console.log('[Vango] Applying', patches.length, 'patches (seq:', seq, ')');
        }

        // Clear any pending optimistic updates that server confirmed
        this.optimistic.clearPending();

        // Apply patches to DOM
        this.patchApplier.apply(patches);

        // Re-initialize hooks on new elements
        this.hooks.updateFromDOM();
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
            case ControlType.RESYNC:
                this._handleResync();
                break;
            case ControlType.CLOSE:
                // Server requesting close
                this.wsManager.close();
                break;
        }
    }

    /**
     * Handle resync (full page refresh needed)
     */
    _handleResync() {
        if (this.options.debug) {
            console.log('[Vango] Resync requested, reloading page');
        }
        location.reload();
    }

    /**
     * Handle server error
     */
    _handleServerError(buffer) {
        // Error format: [code:2][fatal:1][message...]
        if (buffer.length < 3) return;

        const code = (buffer[0] << 8) | buffer[1];
        const fatal = buffer[2] === 1;
        const messageBytes = buffer.slice(3);
        const { value: message } = this.codec.decodeString(messageBytes, 0);

        const errorMessages = {
            0x0001: 'Session expired',
            0x0002: 'Invalid event',
            0x0003: 'Rate limited',
            0x0004: 'Server error',
            0x0005: 'Handler panic',
            0x0006: 'Invalid CSRF',
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

        // Wrap in event frame
        const frame = new Uint8Array(1 + eventBuffer.length);
        frame[0] = FrameType.EVENT;
        frame.set(eventBuffer, 1);

        this.wsManager.send(frame);

        if (this.options.debug) {
            console.log('[Vango] Sent event:', { type, hid, data, seq: this.seq });
        }
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
     * Disconnect and cleanup
     */
    destroy() {
        this.eventCapture.detach();
        this.hooks.destroyAll();
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
export default VangoClient;
