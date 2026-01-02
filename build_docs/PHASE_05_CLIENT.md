# Phase 5: Thin Client ✅ VERIFIED

> **The minimal JavaScript runtime that brings server-rendered pages to life**

**Status**: ✅ VERIFIED (2026-01-02)

## Verification Summary

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| Bundle size (gzipped) | ~15KB | 15.90 KB | ✅ |
| Unit tests | Pass | 48 tests passing | ✅ |
| Standard hooks (Spec §8.4) | 7 hooks | 7 implemented | ✅ |
| VangoUI helper hooks | - | 4 implemented | ✅ |
| Protocol compatibility | Match Go server | Verified | ✅ |

### Standard Hooks (Spec Section 8.4)
- ✅ Sortable - Drag-to-reorder lists
- ✅ Draggable - Free-form element dragging
- ✅ Droppable - Drop zones for draggables
- ✅ Resizable - Resize handles on elements
- ✅ Tooltip - Hover tooltips
- ✅ Dropdown - Click-outside-to-close
- ✅ Collapsible - Expand/collapse animation

### VangoUI Helper Hooks
- ✅ FocusTrap - Modal accessibility
- ✅ Portal - DOM repositioning for z-index
- ✅ Dialog - Modal dialog behavior
- ✅ Popover - Floating content positioning

---

---

## Overview

The thin client is a lightweight JavaScript runtime (~12-15KB gzipped) that:

1. Establishes WebSocket connection to the server
2. Captures user events and sends them to the server
3. Receives binary patches and applies them to the DOM
4. Handles reconnection gracefully
5. Optionally provides optimistic updates for instant feedback
6. Manages client hooks for 60fps interactions

### Design Principles

1. **Minimal Size**: Every byte counts. No dependencies.
2. **Progressive Enhancement**: Page works without JS, enhanced with it.
3. **Resilient**: Handles disconnection, reconnection, errors gracefully.
4. **Fast**: Binary protocol, efficient DOM operations.
5. **Simple**: Easy to understand, debug, and extend.

### Size Budget

| Component | Target Size (gzip) |
|-----------|-------------------|
| Core (WebSocket, patches) | 6 KB |
| Event capture | 2 KB |
| Reconnection logic | 1 KB |
| Optimistic updates | 1 KB |
| Standard hooks | 3 KB |
| **Total** | **13 KB** |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Thin Client                             │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                     VangoClient                           │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐    │  │
│  │  │  WebSocket  │  │   Event     │  │     Patch       │    │  │
│  │  │  Manager    │←→│   Capture   │  │     Applier     │    │  │
│  │  └──────┬──────┘  └──────┬──────┘  └────────┬────────┘    │  │
│  │         │                │                   │            │  │
│  │         ▼                ▼                   ▼            │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐    │  │
│  │  │   Binary    │  │  Optimistic │  │      Hook       │    │  │
│  │  │   Codec     │  │   Updates   │  │     Manager     │    │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘    │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                  │
│                              ▼                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                        DOM                                │  │
│  │   Elements with data-hid attributes                       │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Module Structure

```
client/
├── src/
│   ├── index.js           # Entry point, VangoClient class
│   ├── websocket.js       # WebSocket connection management
│   ├── codec.js           # Binary encoding/decoding
│   ├── events.js          # Event capture and encoding
│   ├── patches.js         # Patch application
│   ├── optimistic.js      # Optimistic update handling
│   ├── connection.js      # Connection state management, toast notifications
│   ├── url.js             # URL/history management
│   ├── prefs.js           # Client preferences (localStorage sync)
│   ├── utils.js           # Helper functions
│   └── hooks/
│       ├── manager.js     # Hook lifecycle management
│       ├── sortable.js    # Drag-to-reorder lists (Spec §8.4)
│       ├── draggable.js   # Free-form element dragging (Spec §8.4)
│       ├── droppable.js   # Drop zones (Spec §8.4)
│       ├── resizable.js   # Resize handles (Spec §8.4)
│       ├── tooltip.js     # Hover tooltips (Spec §8.4)
│       ├── dropdown.js    # Click-outside-to-close (Spec §8.4)
│       ├── collapsible.js # Expand/collapse animation (Spec §8.4)
│       ├── focustrap.js   # Modal accessibility (VangoUI)
│       ├── portal.js      # DOM repositioning (VangoUI)
│       ├── dialog.js      # Modal dialog behavior (VangoUI)
│       └── popover.js     # Floating content (VangoUI)
├── test/
│   ├── codec.test.js      # 30 codec unit tests
│   └── integration.test.js # 18 integration tests
├── dist/
│   ├── vango.js           # Development build
│   └── vango.min.js       # Production build (15.90 KB gzipped)
├── build.js               # Build script (esbuild)
└── package.json
```

---

## Core Implementation

### Entry Point (index.js)

```javascript
/**
 * Vango Thin Client
 *
 * Minimal runtime for server-driven web applications.
 * Connects to server via WebSocket, captures events, applies patches.
 */

import { WebSocketManager } from './websocket.js';
import { EventCapture } from './events.js';
import { PatchApplier } from './patches.js';
import { BinaryCodec } from './codec.js';
import { OptimisticUpdates } from './optimistic.js';
import { HookManager } from './hooks/manager.js';

class VangoClient {
    constructor(options = {}) {
        this.options = {
            wsUrl: options.wsUrl || this._defaultWsUrl(),
            reconnect: options.reconnect !== false,
            reconnectInterval: options.reconnectInterval || 1000,
            reconnectMaxInterval: options.reconnectMaxInterval || 30000,
            heartbeatInterval: options.heartbeatInterval || 30000,
            debug: options.debug || false,
            ...options
        };

        // Core components
        this.codec = new BinaryCodec();
        this.nodeMap = new Map();  // hid -> DOM node
        this.ws = null;
        this.connected = false;
        this.sessionId = null;
        this.seq = 0;

        // Sub-systems
        this.patchApplier = new PatchApplier(this);
        this.eventCapture = new EventCapture(this);
        this.optimistic = new OptimisticUpdates(this);
        this.hooks = new HookManager(this);

        // Callbacks
        this.onConnect = options.onConnect || (() => {});
        this.onDisconnect = options.onDisconnect || (() => {});
        this.onError = options.onError || ((err) => console.error('[Vango]', err));

        // Initialize
        this._buildNodeMap();
        this._connect();
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
     * Establish WebSocket connection
     */
    _connect() {
        if (this.ws) {
            this.ws.close();
        }

        this.ws = new WebSocket(this.options.wsUrl);
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
        if (this.options.debug) {
            console.log('[Vango] WebSocket connected');
        }

        // Send handshake
        this._sendHandshake();

        // Start heartbeat
        this._startHeartbeat();
    }

    /**
     * Send handshake message
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
                height: window.innerHeight
            }
        };

        // Handshake is JSON (only message that's not binary)
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
                    this.onConnect();

                    if (this.options.debug) {
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
            this._handleBinaryMessage(new Uint8Array(event.data));
        }
    }

    /**
     * Handle binary message (patches or control)
     */
    _handleBinaryMessage(buffer) {
        const frameType = buffer[0];
        const payload = buffer.slice(1);

        switch (frameType) {
            case 0x00: // Patches
                this._handlePatches(payload);
                break;
            case 0x01: // Control (pong, resync, etc.)
                this._handleControl(payload);
                break;
            default:
                console.warn('[Vango] Unknown frame type:', frameType);
        }
    }

    /**
     * Handle patch frame
     */
    _handlePatches(buffer) {
        const patches = this.codec.decodePatches(buffer);

        if (this.options.debug) {
            console.log('[Vango] Applying', patches.length, 'patches');
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
        const controlType = buffer[0];

        switch (controlType) {
            case 0x00: // Pong
                // Heartbeat acknowledged
                break;
            case 0x01: // Resync required
                this._handleResync();
                break;
            case 0x02: // Error
                const errorCode = buffer[1];
                this._handleServerError(errorCode);
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
    _handleServerError(code) {
        const messages = {
            0x01: 'Session expired',
            0x02: 'Invalid event',
            0x03: 'Rate limited',
            0x04: 'Server error'
        };

        this.onError(new Error(messages[code] || `Unknown error: ${code}`));
    }

    /**
     * Handle WebSocket close
     */
    _onClose(event) {
        this.connected = false;
        this._stopHeartbeat();

        if (this.options.debug) {
            console.log('[Vango] WebSocket closed:', event.code, event.reason);
        }

        this.onDisconnect();

        // Reconnect if enabled
        if (this.options.reconnect && !event.wasClean) {
            this._scheduleReconnect();
        }
    }

    /**
     * Handle WebSocket error
     */
    _onError(event) {
        this.onError(new Error('WebSocket error'));
    }

    /**
     * Schedule reconnection with exponential backoff
     */
    _scheduleReconnect() {
        const delay = Math.min(
            this.options.reconnectInterval * Math.pow(2, this._reconnectAttempts || 0),
            this.options.reconnectMaxInterval
        );

        this._reconnectAttempts = (this._reconnectAttempts || 0) + 1;

        if (this.options.debug) {
            console.log(`[Vango] Reconnecting in ${delay}ms (attempt ${this._reconnectAttempts})`);
        }

        setTimeout(() => this._connect(), delay);
    }

    /**
     * Start heartbeat timer
     */
    _startHeartbeat() {
        this._heartbeatTimer = setInterval(() => {
            if (this.connected) {
                this._sendPing();
            }
        }, this.options.heartbeatInterval);
    }

    /**
     * Stop heartbeat timer
     */
    _stopHeartbeat() {
        if (this._heartbeatTimer) {
            clearInterval(this._heartbeatTimer);
            this._heartbeatTimer = null;
        }
    }

    /**
     * Send ping (control message)
     */
    _sendPing() {
        const buffer = new Uint8Array([0x01, 0x00]); // Control frame, Ping
        this.ws.send(buffer);
    }

    /**
     * Send event to server
     */
    sendEvent(type, hid, data = null) {
        if (!this.connected) {
            if (this.options.debug) {
                console.log('[Vango] Not connected, queuing event');
            }
            // Could queue events here for later sending
            return;
        }

        const buffer = this.codec.encodeEvent(type, hid, data);
        this.ws.send(buffer);

        if (this.options.debug) {
            console.log('[Vango] Sent event:', { type, hid, data });
        }
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
     * Disconnect and cleanup
     */
    destroy() {
        this._stopHeartbeat();
        this.eventCapture.detach();
        this.hooks.destroyAll();

        if (this.ws) {
            this.options.reconnect = false;
            this.ws.close();
        }
    }
}

// Auto-initialize on DOM ready
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
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}

// Export for manual initialization
export { VangoClient };
export default VangoClient;
```

---

### Binary Codec (codec.js)

```javascript
/**
 * Binary encoding/decoding for Vango protocol
 */

// Event type constants (must match server)
export const EventType = {
    CLICK: 0x01,
    DBLCLICK: 0x02,
    INPUT: 0x03,
    CHANGE: 0x04,
    SUBMIT: 0x05,
    FOCUS: 0x06,
    BLUR: 0x07,
    KEYDOWN: 0x08,
    KEYUP: 0x09,
    MOUSEENTER: 0x0A,
    MOUSELEAVE: 0x0B,
    SCROLL: 0x0C,
    HOOK: 0x0D,
    NAVIGATE: 0x0E
};

// Patch type constants (must match server)
export const PatchType = {
    SET_TEXT: 0x01,
    SET_ATTR: 0x02,
    REMOVE_ATTR: 0x03,
    ADD_CLASS: 0x04,
    REMOVE_CLASS: 0x05,
    SET_STYLE: 0x06,
    INSERT_BEFORE: 0x07,
    INSERT_AFTER: 0x08,
    APPEND_CHILD: 0x09,
    REMOVE_NODE: 0x0A,
    REPLACE_NODE: 0x0B,
    SET_VALUE: 0x0C,
    SET_CHECKED: 0x0D,
    SET_SELECTED: 0x0E,
    FOCUS: 0x0F,
    BLUR: 0x10,
    SCROLL_TO: 0x11
};

// Key modifier flags
export const KeyMod = {
    CTRL: 0x01,
    SHIFT: 0x02,
    ALT: 0x04,
    META: 0x08
};

export class BinaryCodec {
    constructor() {
        this.textEncoder = new TextEncoder();
        this.textDecoder = new TextDecoder();
    }

    /**
     * Encode event to binary buffer
     */
    encodeEvent(type, hid, data) {
        const parts = [];

        // Frame type (0x00 = event)
        parts.push(new Uint8Array([0x00]));

        // Event type
        parts.push(new Uint8Array([type]));

        // HID (varint encoded)
        parts.push(this.encodeVarint(this.hidToInt(hid)));

        // Payload (depends on event type)
        switch (type) {
            case EventType.INPUT:
            case EventType.CHANGE:
                parts.push(this.encodeString(data.value));
                break;

            case EventType.SUBMIT:
                parts.push(this.encodeFormData(data));
                break;

            case EventType.KEYDOWN:
            case EventType.KEYUP:
                parts.push(this.encodeKeyEvent(data));
                break;

            case EventType.SCROLL:
                parts.push(this.encodeScrollEvent(data));
                break;

            case EventType.HOOK:
                parts.push(this.encodeHookEvent(data));
                break;

            case EventType.NAVIGATE:
                parts.push(this.encodeString(data.path));
                break;

            // Click, focus, blur, etc. have no payload
        }

        return this.concat(parts);
    }

    /**
     * Decode patches from binary buffer
     */
    decodePatches(buffer) {
        const patches = [];
        let offset = 0;

        // Read patch count
        const { value: count, bytesRead } = this.decodeVarint(buffer, offset);
        offset += bytesRead;

        // Read each patch
        for (let i = 0; i < count; i++) {
            const { patch, bytesRead } = this.decodePatch(buffer, offset);
            patches.push(patch);
            offset += bytesRead;
        }

        return patches;
    }

    /**
     * Decode single patch
     */
    decodePatch(buffer, offset) {
        const startOffset = offset;
        const patch = {};

        // Patch type
        patch.type = buffer[offset++];

        // Target HID
        const { value: hidInt, bytesRead: hidBytes } = this.decodeVarint(buffer, offset);
        patch.hid = this.intToHid(hidInt);
        offset += hidBytes;

        // Payload (depends on patch type)
        switch (patch.type) {
            case PatchType.SET_TEXT:
            case PatchType.SET_VALUE: {
                const { value, bytesRead } = this.decodeString(buffer, offset);
                patch.text = value;
                offset += bytesRead;
                break;
            }

            case PatchType.SET_ATTR:
            case PatchType.SET_STYLE: {
                const { value: key, bytesRead: keyBytes } = this.decodeString(buffer, offset);
                offset += keyBytes;
                const { value: val, bytesRead: valBytes } = this.decodeString(buffer, offset);
                offset += valBytes;
                patch.key = key;
                patch.value = val;
                break;
            }

            case PatchType.REMOVE_ATTR: {
                const { value, bytesRead } = this.decodeString(buffer, offset);
                patch.key = value;
                offset += bytesRead;
                break;
            }

            case PatchType.ADD_CLASS:
            case PatchType.REMOVE_CLASS: {
                const { value, bytesRead } = this.decodeString(buffer, offset);
                patch.className = value;
                offset += bytesRead;
                break;
            }

            case PatchType.INSERT_BEFORE:
            case PatchType.INSERT_AFTER:
            case PatchType.APPEND_CHILD:
            case PatchType.REPLACE_NODE: {
                const { value: refHid, bytesRead: refBytes } = this.decodeVarint(buffer, offset);
                patch.refHid = this.intToHid(refHid);
                offset += refBytes;
                const { vnode, bytesRead: vnodeBytes } = this.decodeVNode(buffer, offset);
                patch.vnode = vnode;
                offset += vnodeBytes;
                break;
            }

            case PatchType.REMOVE_NODE:
            case PatchType.FOCUS:
            case PatchType.BLUR:
                // No payload
                break;

            case PatchType.SET_CHECKED:
            case PatchType.SET_SELECTED: {
                patch.value = buffer[offset++] === 1;
                break;
            }

            case PatchType.SCROLL_TO: {
                const { value: x, bytesRead: xBytes } = this.decodeInt32(buffer, offset);
                offset += xBytes;
                const { value: y, bytesRead: yBytes } = this.decodeInt32(buffer, offset);
                offset += yBytes;
                patch.x = x;
                patch.y = y;
                break;
            }
        }

        return { patch, bytesRead: offset - startOffset };
    }

    /**
     * Decode VNode from buffer
     */
    decodeVNode(buffer, offset) {
        const startOffset = offset;
        const vnode = {};

        // Node type
        const nodeType = buffer[offset++];

        switch (nodeType) {
            case 0x01: // Element
                vnode.type = 'element';

                // Tag
                const { value: tag, bytesRead: tagBytes } = this.decodeString(buffer, offset);
                vnode.tag = tag;
                offset += tagBytes;

                // HID (0 = no hid)
                const { value: hidInt, bytesRead: hidBytes } = this.decodeVarint(buffer, offset);
                vnode.hid = hidInt === 0 ? null : this.intToHid(hidInt);
                offset += hidBytes;

                // Attributes
                const { value: attrCount, bytesRead: attrCountBytes } = this.decodeVarint(buffer, offset);
                offset += attrCountBytes;
                vnode.attrs = {};

                for (let i = 0; i < attrCount; i++) {
                    const { value: key, bytesRead: keyBytes } = this.decodeString(buffer, offset);
                    offset += keyBytes;
                    const { value: val, bytesRead: valBytes } = this.decodeString(buffer, offset);
                    offset += valBytes;
                    vnode.attrs[key] = val;
                }

                // Children
                const { value: childCount, bytesRead: childCountBytes } = this.decodeVarint(buffer, offset);
                offset += childCountBytes;
                vnode.children = [];

                for (let i = 0; i < childCount; i++) {
                    const { vnode: child, bytesRead: childBytes } = this.decodeVNode(buffer, offset);
                    vnode.children.push(child);
                    offset += childBytes;
                }
                break;

            case 0x02: // Text
                vnode.type = 'text';
                const { value: text, bytesRead: textBytes } = this.decodeString(buffer, offset);
                vnode.text = text;
                offset += textBytes;
                break;

            case 0x03: // Fragment
                vnode.type = 'fragment';
                const { value: fragChildCount, bytesRead: fragCountBytes } = this.decodeVarint(buffer, offset);
                offset += fragCountBytes;
                vnode.children = [];

                for (let i = 0; i < fragChildCount; i++) {
                    const { vnode: child, bytesRead: childBytes } = this.decodeVNode(buffer, offset);
                    vnode.children.push(child);
                    offset += childBytes;
                }
                break;
        }

        return { vnode, bytesRead: offset - startOffset };
    }

    /**
     * Encode string with length prefix
     */
    encodeString(str) {
        const bytes = this.textEncoder.encode(str);
        const length = this.encodeVarint(bytes.length);
        return this.concat([length, bytes]);
    }

    /**
     * Decode string with length prefix
     */
    decodeString(buffer, offset) {
        const { value: length, bytesRead: lengthBytes } = this.decodeVarint(buffer, offset);
        const strBytes = buffer.slice(offset + lengthBytes, offset + lengthBytes + length);
        const value = this.textDecoder.decode(strBytes);
        return { value, bytesRead: lengthBytes + length };
    }

    /**
     * Encode varint (unsigned variable-length integer)
     */
    encodeVarint(value) {
        const bytes = [];
        while (value > 0x7F) {
            bytes.push((value & 0x7F) | 0x80);
            value >>>= 7;
        }
        bytes.push(value & 0x7F);
        return new Uint8Array(bytes);
    }

    /**
     * Decode varint
     */
    decodeVarint(buffer, offset) {
        let value = 0;
        let shift = 0;
        let bytesRead = 0;

        while (true) {
            const byte = buffer[offset + bytesRead];
            bytesRead++;
            value |= (byte & 0x7F) << shift;

            if ((byte & 0x80) === 0) {
                break;
            }
            shift += 7;
        }

        return { value, bytesRead };
    }

    /**
     * Encode signed 32-bit integer
     */
    encodeInt32(value) {
        const buffer = new ArrayBuffer(4);
        new DataView(buffer).setInt32(0, value, true); // little-endian
        return new Uint8Array(buffer);
    }

    /**
     * Decode signed 32-bit integer
     */
    decodeInt32(buffer, offset) {
        const view = new DataView(buffer.buffer, buffer.byteOffset + offset, 4);
        return { value: view.getInt32(0, true), bytesRead: 4 };
    }

    /**
     * Encode form data
     */
    encodeFormData(formData) {
        const entries = [];
        for (const [key, value] of formData.entries()) {
            entries.push({ key, value: String(value) });
        }

        const parts = [this.encodeVarint(entries.length)];
        for (const { key, value } of entries) {
            parts.push(this.encodeString(key));
            parts.push(this.encodeString(value));
        }

        return this.concat(parts);
    }

    /**
     * Encode keyboard event
     */
    encodeKeyEvent(data) {
        let modifiers = 0;
        if (data.ctrlKey) modifiers |= KeyMod.CTRL;
        if (data.shiftKey) modifiers |= KeyMod.SHIFT;
        if (data.altKey) modifiers |= KeyMod.ALT;
        if (data.metaKey) modifiers |= KeyMod.META;

        return this.concat([
            this.encodeString(data.key),
            this.encodeString(data.code),
            new Uint8Array([modifiers])
        ]);
    }

    /**
     * Encode scroll event
     */
    encodeScrollEvent(data) {
        return this.concat([
            this.encodeInt32(data.scrollTop),
            this.encodeInt32(data.scrollLeft)
        ]);
    }

    /**
     * Encode hook event
     */
    encodeHookEvent(data) {
        // Hook events are JSON-encoded for flexibility
        const json = JSON.stringify(data);
        return this.encodeString(json);
    }

    /**
     * Convert HID string ("h42") to integer (42)
     */
    hidToInt(hid) {
        return parseInt(hid.slice(1), 10);
    }

    /**
     * Convert integer to HID string
     */
    intToHid(n) {
        return 'h' + n;
    }

    /**
     * Concatenate Uint8Arrays
     */
    concat(arrays) {
        const totalLength = arrays.reduce((sum, arr) => sum + arr.length, 0);
        const result = new Uint8Array(totalLength);
        let offset = 0;

        for (const arr of arrays) {
            result.set(arr, offset);
            offset += arr.length;
        }

        return result;
    }
}
```

---

### Event Capture (events.js)

```javascript
/**
 * Event capture and delegation
 */

import { EventType } from './codec.js';

export class EventCapture {
    constructor(client) {
        this.client = client;
        this.handlers = new Map();
        this.debounceTimers = new Map();
    }

    /**
     * Attach event listeners to document
     */
    attach() {
        // Click events
        this._on('click', this._handleClick.bind(this));
        this._on('dblclick', this._handleDblClick.bind(this));

        // Input events (debounced)
        this._on('input', this._handleInput.bind(this));
        this._on('change', this._handleChange.bind(this));

        // Form events
        this._on('submit', this._handleSubmit.bind(this));

        // Focus events
        this._on('focus', this._handleFocus.bind(this), true); // capture
        this._on('blur', this._handleBlur.bind(this), true);   // capture

        // Keyboard events
        this._on('keydown', this._handleKeyDown.bind(this));
        this._on('keyup', this._handleKeyUp.bind(this));

        // Mouse events
        this._on('mouseenter', this._handleMouseEnter.bind(this), true);
        this._on('mouseleave', this._handleMouseLeave.bind(this), true);

        // Scroll events (throttled)
        this._on('scroll', this._handleScroll.bind(this), true);

        // Navigation
        this._on('click', this._handleLinkClick.bind(this));
        window.addEventListener('popstate', this._handlePopState.bind(this));
    }

    /**
     * Detach all event listeners
     */
    detach() {
        for (const [eventType, handler] of this.handlers) {
            document.removeEventListener(eventType, handler);
        }
        this.handlers.clear();

        for (const timer of this.debounceTimers.values()) {
            clearTimeout(timer);
        }
        this.debounceTimers.clear();
    }

    /**
     * Helper to add event listener
     */
    _on(eventType, handler, capture = false) {
        document.addEventListener(eventType, handler, { capture, passive: false });
        this.handlers.set(eventType, handler);
    }

    /**
     * Find closest element with data-hid
     */
    _findHidElement(target) {
        return target.closest('[data-hid]');
    }

    /**
     * Handle click event
     */
    _handleClick(event) {
        const el = this._findHidElement(event.target);
        if (!el) return;

        // Check if this element has click handler (data-onclick attribute)
        if (!el.hasAttribute('data-on-click')) return;

        event.preventDefault();

        // Apply optimistic updates if configured
        this.client.optimistic.applyOptimistic(el, 'click');

        // Send event
        this.client.sendEvent(EventType.CLICK, el.dataset.hid);
    }

    /**
     * Handle double-click event
     */
    _handleDblClick(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-dblclick')) return;

        event.preventDefault();
        this.client.sendEvent(EventType.DBLCLICK, el.dataset.hid);
    }

    /**
     * Handle input event (debounced)
     */
    _handleInput(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-input')) return;

        const hid = el.dataset.hid;
        const debounceMs = parseInt(el.dataset.debounce || '100', 10);

        // Clear existing timer
        if (this.debounceTimers.has(hid)) {
            clearTimeout(this.debounceTimers.get(hid));
        }

        // Set new timer
        const timer = setTimeout(() => {
            this.debounceTimers.delete(hid);
            this.client.sendEvent(EventType.INPUT, hid, { value: el.value });
        }, debounceMs);

        this.debounceTimers.set(hid, timer);
    }

    /**
     * Handle change event
     */
    _handleChange(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-change')) return;

        let value;
        if (el.type === 'checkbox') {
            value = el.checked ? 'true' : 'false';
        } else if (el.type === 'radio') {
            value = el.checked ? el.value : '';
        } else if (el.tagName === 'SELECT' && el.multiple) {
            value = Array.from(el.selectedOptions).map(o => o.value).join(',');
        } else {
            value = el.value;
        }

        // Apply optimistic updates
        this.client.optimistic.applyOptimistic(el, 'change');

        this.client.sendEvent(EventType.CHANGE, el.dataset.hid, { value });
    }

    /**
     * Handle form submit
     */
    _handleSubmit(event) {
        const form = event.target.closest('form[data-hid]');
        if (!form || !form.hasAttribute('data-on-submit')) return;

        event.preventDefault();

        const formData = new FormData(form);
        this.client.sendEvent(EventType.SUBMIT, form.dataset.hid, formData);
    }

    /**
     * Handle focus event
     */
    _handleFocus(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-focus')) return;

        this.client.sendEvent(EventType.FOCUS, el.dataset.hid);
    }

    /**
     * Handle blur event
     */
    _handleBlur(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-blur')) return;

        this.client.sendEvent(EventType.BLUR, el.dataset.hid);
    }

    /**
     * Handle keydown event
     */
    _handleKeyDown(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-keydown')) return;

        // Check for specific key filter
        const keyFilter = el.dataset.keyFilter;
        if (keyFilter && !this._matchesKeyFilter(event, keyFilter)) {
            return;
        }

        // Some key combinations should not prevent default
        const shouldPrevent = el.dataset.preventDefault !== 'false';
        if (shouldPrevent) {
            event.preventDefault();
        }

        this.client.sendEvent(EventType.KEYDOWN, el.dataset.hid, {
            key: event.key,
            code: event.code,
            ctrlKey: event.ctrlKey,
            shiftKey: event.shiftKey,
            altKey: event.altKey,
            metaKey: event.metaKey
        });
    }

    /**
     * Handle keyup event
     */
    _handleKeyUp(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-keyup')) return;

        this.client.sendEvent(EventType.KEYUP, el.dataset.hid, {
            key: event.key,
            code: event.code,
            ctrlKey: event.ctrlKey,
            shiftKey: event.shiftKey,
            altKey: event.altKey,
            metaKey: event.metaKey
        });
    }

    /**
     * Handle mouseenter event
     */
    _handleMouseEnter(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-mouseenter')) return;

        this.client.sendEvent(EventType.MOUSEENTER, el.dataset.hid);
    }

    /**
     * Handle mouseleave event
     */
    _handleMouseLeave(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-mouseleave')) return;

        this.client.sendEvent(EventType.MOUSELEAVE, el.dataset.hid);
    }

    /**
     * Handle scroll event (throttled)
     */
    _handleScroll(event) {
        const el = this._findHidElement(event.target);
        if (!el || !el.hasAttribute('data-on-scroll')) return;

        const hid = el.dataset.hid;
        const throttleMs = parseInt(el.dataset.throttle || '100', 10);

        // Simple throttle
        if (this._scrollThrottled && this._scrollThrottled.has(hid)) {
            return;
        }

        if (!this._scrollThrottled) {
            this._scrollThrottled = new Set();
        }

        this._scrollThrottled.add(hid);

        setTimeout(() => {
            this._scrollThrottled.delete(hid);
        }, throttleMs);

        this.client.sendEvent(EventType.SCROLL, hid, {
            scrollTop: el.scrollTop,
            scrollLeft: el.scrollLeft
        });
    }

    /**
     * Handle link click for client-side navigation
     */
    _handleLinkClick(event) {
        const link = event.target.closest('a[href]');
        if (!link) return;

        // Check if this is an internal link
        const href = link.getAttribute('href');
        if (!href || href.startsWith('http') || href.startsWith('//')) {
            return; // External link, let browser handle
        }

        if (link.hasAttribute('data-external') || link.target === '_blank') {
            return; // Explicitly external
        }

        // Prevent default and navigate via WebSocket
        event.preventDefault();

        // Update browser URL
        history.pushState(null, '', href);

        // Send navigate event to server
        this.client.sendEvent(EventType.NAVIGATE, 'nav', { path: href });
    }

    /**
     * Handle browser back/forward
     */
    _handlePopState(event) {
        this.client.sendEvent(EventType.NAVIGATE, 'nav', { path: location.pathname });
    }

    /**
     * Check if event matches key filter
     * Format: "Enter" or "Ctrl+s" or "Meta+Enter"
     */
    _matchesKeyFilter(event, filter) {
        const parts = filter.split('+');
        const key = parts.pop().toLowerCase();
        const modifiers = new Set(parts.map(m => m.toLowerCase()));

        // Check key
        if (event.key.toLowerCase() !== key) {
            return false;
        }

        // Check modifiers
        const hasCtrl = modifiers.has('ctrl') || modifiers.has('control');
        const hasShift = modifiers.has('shift');
        const hasAlt = modifiers.has('alt');
        const hasMeta = modifiers.has('meta') || modifiers.has('cmd');

        if (hasCtrl !== event.ctrlKey) return false;
        if (hasShift !== event.shiftKey) return false;
        if (hasAlt !== event.altKey) return false;
        if (hasMeta !== event.metaKey) return false;

        return true;
    }
}
```

---

### Patch Applier (patches.js)

```javascript
/**
 * Apply patches to the DOM
 */

import { PatchType } from './codec.js';

export class PatchApplier {
    constructor(client) {
        this.client = client;
    }

    /**
     * Apply array of patches
     */
    apply(patches) {
        for (const patch of patches) {
            this.applyPatch(patch);
        }
    }

    /**
     * Apply single patch
     */
    applyPatch(patch) {
        const el = this.client.getNode(patch.hid);

        if (!el && patch.type !== PatchType.INSERT_BEFORE &&
            patch.type !== PatchType.INSERT_AFTER &&
            patch.type !== PatchType.APPEND_CHILD) {
            console.warn('[Vango] Node not found:', patch.hid);
            return;
        }

        switch (patch.type) {
            case PatchType.SET_TEXT:
                this._setText(el, patch.text);
                break;

            case PatchType.SET_ATTR:
                this._setAttr(el, patch.key, patch.value);
                break;

            case PatchType.REMOVE_ATTR:
                this._removeAttr(el, patch.key);
                break;

            case PatchType.ADD_CLASS:
                el.classList.add(patch.className);
                break;

            case PatchType.REMOVE_CLASS:
                el.classList.remove(patch.className);
                break;

            case PatchType.SET_STYLE:
                el.style[patch.key] = patch.value;
                break;

            case PatchType.INSERT_BEFORE:
                this._insertBefore(patch.refHid, patch.vnode);
                break;

            case PatchType.INSERT_AFTER:
                this._insertAfter(patch.refHid, patch.vnode);
                break;

            case PatchType.APPEND_CHILD:
                this._appendChild(patch.hid, patch.vnode);
                break;

            case PatchType.REMOVE_NODE:
                this._removeNode(el, patch.hid);
                break;

            case PatchType.REPLACE_NODE:
                this._replaceNode(el, patch.hid, patch.vnode);
                break;

            case PatchType.SET_VALUE:
                this._setValue(el, patch.text);
                break;

            case PatchType.SET_CHECKED:
                el.checked = patch.value;
                break;

            case PatchType.SET_SELECTED:
                el.selected = patch.value;
                break;

            case PatchType.FOCUS:
                el.focus();
                break;

            case PatchType.BLUR:
                el.blur();
                break;

            case PatchType.SCROLL_TO:
                el.scrollTo({ left: patch.x, top: patch.y, behavior: 'smooth' });
                break;
        }
    }

    /**
     * Set text content
     */
    _setText(el, text) {
        el.textContent = text;
    }

    /**
     * Set attribute with special cases
     */
    _setAttr(el, key, value) {
        // Special attribute handling
        switch (key) {
            case 'class':
                el.className = value;
                break;
            case 'for':
                el.htmlFor = value;
                break;
            case 'value':
                this._setValue(el, value);
                break;
            case 'checked':
                el.checked = value === 'true' || value === '';
                break;
            case 'selected':
                el.selected = value === 'true' || value === '';
                break;
            case 'disabled':
            case 'readonly':
            case 'required':
            case 'multiple':
            case 'autofocus':
                // Boolean attributes
                if (value === 'true' || value === '') {
                    el.setAttribute(key, '');
                } else {
                    el.removeAttribute(key);
                }
                break;
            default:
                // Data attributes for event handlers
                if (key.startsWith('data-on-')) {
                    el.setAttribute(key, value);
                } else {
                    el.setAttribute(key, value);
                }
        }
    }

    /**
     * Remove attribute
     */
    _removeAttr(el, key) {
        switch (key) {
            case 'class':
                el.className = '';
                break;
            case 'for':
                el.htmlFor = '';
                break;
            case 'value':
                el.value = '';
                break;
            case 'checked':
                el.checked = false;
                break;
            case 'selected':
                el.selected = false;
                break;
            default:
                el.removeAttribute(key);
        }
    }

    /**
     * Set input value (preserving cursor position)
     */
    _setValue(el, value) {
        if (el.value === value) return;

        // Preserve cursor position for text inputs
        if (el.type === 'text' || el.type === 'textarea') {
            const start = el.selectionStart;
            const end = el.selectionEnd;
            el.value = value;

            // Restore cursor if element is focused
            if (document.activeElement === el) {
                el.setSelectionRange(
                    Math.min(start, value.length),
                    Math.min(end, value.length)
                );
            }
        } else {
            el.value = value;
        }
    }

    /**
     * Insert node before reference
     */
    _insertBefore(refHid, vnode) {
        const refEl = this.client.getNode(refHid);
        if (!refEl) {
            console.warn('[Vango] Reference node not found:', refHid);
            return;
        }

        const newEl = this._createNode(vnode);
        refEl.parentNode.insertBefore(newEl, refEl);
    }

    /**
     * Insert node after reference
     */
    _insertAfter(refHid, vnode) {
        const refEl = this.client.getNode(refHid);
        if (!refEl) {
            console.warn('[Vango] Reference node not found:', refHid);
            return;
        }

        const newEl = this._createNode(vnode);
        refEl.parentNode.insertBefore(newEl, refEl.nextSibling);
    }

    /**
     * Append child to parent
     */
    _appendChild(parentHid, vnode) {
        const parentEl = this.client.getNode(parentHid);
        if (!parentEl) {
            console.warn('[Vango] Parent node not found:', parentHid);
            return;
        }

        const newEl = this._createNode(vnode);
        parentEl.appendChild(newEl);
    }

    /**
     * Remove node
     */
    _removeNode(el, hid) {
        // Cleanup hooks
        this.client.hooks.destroyForNode(el);

        // Remove from map
        this.client.unregisterNode(hid);

        // Unregister all children with HIDs
        el.querySelectorAll('[data-hid]').forEach(child => {
            this.client.hooks.destroyForNode(child);
            this.client.unregisterNode(child.dataset.hid);
        });

        // Remove from DOM
        el.remove();
    }

    /**
     * Replace node
     */
    _replaceNode(el, hid, vnode) {
        // Cleanup old
        this.client.hooks.destroyForNode(el);
        this.client.unregisterNode(hid);
        el.querySelectorAll('[data-hid]').forEach(child => {
            this.client.hooks.destroyForNode(child);
            this.client.unregisterNode(child.dataset.hid);
        });

        // Create and insert new
        const newEl = this._createNode(vnode);
        el.replaceWith(newEl);
    }

    /**
     * Create DOM node from VNode
     */
    _createNode(vnode) {
        switch (vnode.type) {
            case 'element':
                return this._createElement(vnode);
            case 'text':
                return document.createTextNode(vnode.text);
            case 'fragment':
                return this._createFragment(vnode);
            default:
                console.warn('[Vango] Unknown vnode type:', vnode.type);
                return document.createTextNode('');
        }
    }

    /**
     * Create element from VNode
     */
    _createElement(vnode) {
        const el = document.createElement(vnode.tag);

        // Set attributes
        for (const [key, value] of Object.entries(vnode.attrs || {})) {
            this._setAttr(el, key, value);
        }

        // Set HID and register
        if (vnode.hid) {
            el.dataset.hid = vnode.hid;
            this.client.registerNode(vnode.hid, el);
        }

        // Create children
        for (const child of vnode.children || []) {
            el.appendChild(this._createNode(child));
        }

        // Initialize hooks
        this.client.hooks.initializeForNode(el);

        return el;
    }

    /**
     * Create document fragment
     */
    _createFragment(vnode) {
        const frag = document.createDocumentFragment();
        for (const child of vnode.children || []) {
            frag.appendChild(this._createNode(child));
        }
        return frag;
    }
}
```

---

### Optimistic Updates (optimistic.js)

```javascript
/**
 * Optimistic update handling for instant feedback
 */

export class OptimisticUpdates {
    constructor(client) {
        this.client = client;
        this.pending = new Map(); // hid -> { original, applied }
    }

    /**
     * Apply optimistic update based on element data attributes
     */
    applyOptimistic(el, eventType) {
        const hid = el.dataset.hid;

        // Check for optimistic class toggle
        const optimisticClass = el.dataset.optimisticClass;
        if (optimisticClass) {
            this._applyClassOptimistic(el, hid, optimisticClass);
        }

        // Check for optimistic text
        const optimisticText = el.dataset.optimisticText;
        if (optimisticText) {
            this._applyTextOptimistic(el, hid, optimisticText);
        }

        // Check for optimistic attribute
        const optimisticAttr = el.dataset.optimisticAttr;
        const optimisticValue = el.dataset.optimisticValue;
        if (optimisticAttr && optimisticValue !== undefined) {
            this._applyAttrOptimistic(el, hid, optimisticAttr, optimisticValue);
        }

        // Check for parent optimistic class
        const parentOptimisticClass = el.dataset.optimisticParentClass;
        if (parentOptimisticClass && el.parentElement) {
            const parentHid = el.parentElement.dataset.hid || `parent-${hid}`;
            this._applyClassOptimistic(el.parentElement, parentHid, parentOptimisticClass);
        }
    }

    /**
     * Apply optimistic class toggle
     */
    _applyClassOptimistic(el, hid, classConfig) {
        // Format: "classname" or "classname:add" or "classname:remove" or "classname:toggle"
        const [className, action = 'toggle'] = classConfig.split(':');

        // Store original state
        const original = el.classList.contains(className);

        // Apply change
        switch (action) {
            case 'add':
                el.classList.add(className);
                break;
            case 'remove':
                el.classList.remove(className);
                break;
            case 'toggle':
            default:
                el.classList.toggle(className);
        }

        // Track for potential revert
        this._trackPending(hid, 'class', className, original);
    }

    /**
     * Apply optimistic text change
     */
    _applyTextOptimistic(el, hid, text) {
        // Store original
        const original = el.textContent;

        // Apply change
        el.textContent = text;

        // Track for revert
        this._trackPending(hid, 'text', null, original);
    }

    /**
     * Apply optimistic attribute change
     */
    _applyAttrOptimistic(el, hid, attr, value) {
        // Store original
        const original = el.getAttribute(attr);

        // Apply change
        if (value === 'null' || value === '') {
            el.removeAttribute(attr);
        } else {
            el.setAttribute(attr, value);
        }

        // Track for revert
        this._trackPending(hid, 'attr', attr, original);
    }

    /**
     * Track pending optimistic update
     */
    _trackPending(hid, type, key, original) {
        if (!this.pending.has(hid)) {
            this.pending.set(hid, []);
        }
        this.pending.get(hid).push({ type, key, original });
    }

    /**
     * Clear pending updates (called when server confirms)
     */
    clearPending() {
        this.pending.clear();
    }

    /**
     * Revert all pending updates (called on error)
     */
    revertAll() {
        for (const [hid, updates] of this.pending) {
            const el = this.client.getNode(hid);
            if (!el) continue;

            for (const update of updates) {
                this._revertUpdate(el, update);
            }
        }
        this.pending.clear();
    }

    /**
     * Revert single update
     */
    _revertUpdate(el, update) {
        switch (update.type) {
            case 'class':
                if (update.original) {
                    el.classList.add(update.key);
                } else {
                    el.classList.remove(update.key);
                }
                break;
            case 'text':
                el.textContent = update.original;
                break;
            case 'attr':
                if (update.original === null) {
                    el.removeAttribute(update.key);
                } else {
                    el.setAttribute(update.key, update.original);
                }
                break;
        }
    }
}
```

---

### Hook Manager (hooks/manager.js)

```javascript
/**
 * Client hook lifecycle management
 */

import { SortableHook } from './sortable.js';
import { DraggableHook } from './draggable.js';
import { TooltipHook } from './tooltip.js';
import { DropdownHook } from './dropdown.js';

export class HookManager {
    constructor(client) {
        this.client = client;
        this.instances = new Map(); // hid -> { hook, instance }

        // Register standard hooks
        this.hooks = {
            'Sortable': SortableHook,
            'Draggable': DraggableHook,
            'Tooltip': TooltipHook,
            'Dropdown': DropdownHook
        };
    }

    /**
     * Register a custom hook
     */
    register(name, hookClass) {
        this.hooks[name] = hookClass;
    }

    /**
     * Initialize hooks from current DOM
     */
    initializeFromDOM() {
        document.querySelectorAll('[data-hook]').forEach(el => {
            this.initializeForNode(el);
        });
    }

    /**
     * Update hooks after DOM changes
     */
    updateFromDOM() {
        // Initialize any new hooks
        document.querySelectorAll('[data-hook]').forEach(el => {
            const hid = el.dataset.hid;
            if (hid && !this.instances.has(hid)) {
                this.initializeForNode(el);
            }
        });
    }

    /**
     * Initialize hook for a specific node
     */
    initializeForNode(el) {
        const hookName = el.dataset.hook;
        if (!hookName) return;

        const HookClass = this.hooks[hookName];
        if (!HookClass) {
            console.warn('[Vango] Unknown hook:', hookName);
            return;
        }

        const hid = el.dataset.hid;
        if (!hid) {
            console.warn('[Vango] Hook element must have data-hid');
            return;
        }

        // Parse config from data-hook-config
        let config = {};
        try {
            if (el.dataset.hookConfig) {
                config = JSON.parse(el.dataset.hookConfig);
            }
        } catch (e) {
            console.warn('[Vango] Invalid hook config:', e);
        }

        // Create push event function
        const pushEvent = (eventName, data) => {
            this.client.sendEvent(
                0x0D, // HOOK event type
                hid,
                { event: eventName, ...data }
            );
        };

        // Instantiate and mount
        const instance = new HookClass();
        instance.mounted(el, config, pushEvent);

        this.instances.set(hid, { hook: HookClass, instance, el });
    }

    /**
     * Destroy hook for a specific node
     */
    destroyForNode(el) {
        const hid = el.dataset.hid;
        if (!hid) return;

        const entry = this.instances.get(hid);
        if (entry) {
            entry.instance.destroyed(entry.el);
            this.instances.delete(hid);
        }
    }

    /**
     * Destroy all hooks
     */
    destroyAll() {
        for (const [hid, entry] of this.instances) {
            entry.instance.destroyed(entry.el);
        }
        this.instances.clear();
    }

    /**
     * Update hook config
     */
    updateConfig(hid, config) {
        const entry = this.instances.get(hid);
        if (entry) {
            const pushEvent = (eventName, data) => {
                this.client.sendEvent(0x0D, hid, { event: eventName, ...data });
            };
            entry.instance.updated(entry.el, config, pushEvent);
        }
    }
}
```

---

### Sortable Hook (hooks/sortable.js)

```javascript
/**
 * Sortable list hook for drag-to-reorder
 * Uses a minimal custom implementation (~1KB vs 8KB for SortableJS)
 */

export class SortableHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        this.animation = config.animation || 150;
        this.handle = config.handle || null;
        this.ghostClass = config.ghostClass || 'sortable-ghost';
        this.dragClass = config.dragClass || 'sortable-drag';

        this.dragging = null;
        this.ghost = null;
        this.startIndex = -1;

        this._bindEvents();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
    }

    destroyed(el) {
        this._unbindEvents();
    }

    _bindEvents() {
        this._onMouseDown = this._handleMouseDown.bind(this);
        this._onMouseMove = this._handleMouseMove.bind(this);
        this._onMouseUp = this._handleMouseUp.bind(this);

        this.el.addEventListener('mousedown', this._onMouseDown);
        document.addEventListener('mousemove', this._onMouseMove);
        document.addEventListener('mouseup', this._onMouseUp);

        // Touch events
        this._onTouchStart = this._handleTouchStart.bind(this);
        this._onTouchMove = this._handleTouchMove.bind(this);
        this._onTouchEnd = this._handleTouchEnd.bind(this);

        this.el.addEventListener('touchstart', this._onTouchStart, { passive: false });
        document.addEventListener('touchmove', this._onTouchMove, { passive: false });
        document.addEventListener('touchend', this._onTouchEnd);
    }

    _unbindEvents() {
        this.el.removeEventListener('mousedown', this._onMouseDown);
        document.removeEventListener('mousemove', this._onMouseMove);
        document.removeEventListener('mouseup', this._onMouseUp);

        this.el.removeEventListener('touchstart', this._onTouchStart);
        document.removeEventListener('touchmove', this._onTouchMove);
        document.removeEventListener('touchend', this._onTouchEnd);
    }

    _handleMouseDown(e) {
        const item = this._findItem(e.target);
        if (!item) return;

        // Check handle
        if (this.handle && !e.target.closest(this.handle)) return;

        e.preventDefault();
        this._startDrag(item, e.clientY);
    }

    _handleTouchStart(e) {
        const item = this._findItem(e.target);
        if (!item) return;

        if (this.handle && !e.target.closest(this.handle)) return;

        e.preventDefault();
        const touch = e.touches[0];
        this._startDrag(item, touch.clientY);
    }

    _findItem(target) {
        // Find direct child of container
        let item = target;
        while (item && item.parentElement !== this.el) {
            item = item.parentElement;
        }
        return item;
    }

    _startDrag(item, y) {
        this.dragging = item;
        this.startIndex = Array.from(this.el.children).indexOf(item);
        this.startY = y;
        this.itemHeight = item.offsetHeight;

        // Create ghost
        this.ghost = item.cloneNode(true);
        this.ghost.classList.add(this.ghostClass);
        this.ghost.style.position = 'fixed';
        this.ghost.style.zIndex = '9999';
        this.ghost.style.width = `${item.offsetWidth}px`;
        this.ghost.style.left = `${item.getBoundingClientRect().left}px`;
        this.ghost.style.top = `${item.getBoundingClientRect().top}px`;
        this.ghost.style.pointerEvents = 'none';
        this.ghost.style.opacity = '0.8';
        document.body.appendChild(this.ghost);

        // Style dragging item
        item.classList.add(this.dragClass);
        item.style.opacity = '0.4';
    }

    _handleMouseMove(e) {
        if (!this.dragging) return;
        e.preventDefault();
        this._updateDrag(e.clientY);
    }

    _handleTouchMove(e) {
        if (!this.dragging) return;
        e.preventDefault();
        const touch = e.touches[0];
        this._updateDrag(touch.clientY);
    }

    _updateDrag(y) {
        // Move ghost
        const deltaY = y - this.startY;
        const startTop = this.el.children[this.startIndex].getBoundingClientRect().top;
        this.ghost.style.top = `${startTop + deltaY}px`;

        // Find insert position
        const children = Array.from(this.el.children);
        const currentIndex = children.indexOf(this.dragging);

        for (let i = 0; i < children.length; i++) {
            if (i === currentIndex) continue;

            const child = children[i];
            const rect = child.getBoundingClientRect();
            const midpoint = rect.top + rect.height / 2;

            if (y < midpoint && i < currentIndex) {
                this.el.insertBefore(this.dragging, child);
                break;
            } else if (y > midpoint && i > currentIndex) {
                this.el.insertBefore(this.dragging, child.nextSibling);
                break;
            }
        }
    }

    _handleMouseUp(e) {
        if (!this.dragging) return;
        this._endDrag();
    }

    _handleTouchEnd(e) {
        if (!this.dragging) return;
        this._endDrag();
    }

    _endDrag() {
        const endIndex = Array.from(this.el.children).indexOf(this.dragging);

        // Cleanup
        this.dragging.classList.remove(this.dragClass);
        this.dragging.style.opacity = '';
        this.ghost.remove();

        // Only send event if position changed
        if (endIndex !== this.startIndex) {
            const id = this.dragging.dataset.id || this.dragging.dataset.hid;

            this.pushEvent('reorder', {
                id: id,
                fromIndex: this.startIndex,
                toIndex: endIndex
            });
        }

        this.dragging = null;
        this.ghost = null;
    }
}
```

---

### Tooltip Hook (hooks/tooltip.js)

```javascript
/**
 * Simple tooltip hook
 */

export class TooltipHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.tooltip = null;

        this.content = config.content || '';
        this.placement = config.placement || 'top';
        this.delay = config.delay || 200;

        this._bindEvents();
    }

    updated(el, config, pushEvent) {
        this.content = config.content || '';
        this.placement = config.placement || 'top';

        if (this.tooltip) {
            this.tooltip.textContent = this.content;
        }
    }

    destroyed(el) {
        this._unbindEvents();
        this._hideTooltip();
    }

    _bindEvents() {
        this._onMouseEnter = this._handleMouseEnter.bind(this);
        this._onMouseLeave = this._handleMouseLeave.bind(this);

        this.el.addEventListener('mouseenter', this._onMouseEnter);
        this.el.addEventListener('mouseleave', this._onMouseLeave);
    }

    _unbindEvents() {
        this.el.removeEventListener('mouseenter', this._onMouseEnter);
        this.el.removeEventListener('mouseleave', this._onMouseLeave);
    }

    _handleMouseEnter() {
        this._showTimer = setTimeout(() => {
            this._showTooltip();
        }, this.delay);
    }

    _handleMouseLeave() {
        clearTimeout(this._showTimer);
        this._hideTooltip();
    }

    _showTooltip() {
        if (!this.content) return;

        // Create tooltip element
        this.tooltip = document.createElement('div');
        this.tooltip.className = 'vango-tooltip';
        this.tooltip.textContent = this.content;
        this.tooltip.style.cssText = `
            position: fixed;
            z-index: 10000;
            padding: 4px 8px;
            background: #333;
            color: white;
            border-radius: 4px;
            font-size: 12px;
            pointer-events: none;
            white-space: nowrap;
        `;

        document.body.appendChild(this.tooltip);

        // Position
        this._position();
    }

    _position() {
        if (!this.tooltip) return;

        const rect = this.el.getBoundingClientRect();
        const tipRect = this.tooltip.getBoundingClientRect();

        let top, left;

        switch (this.placement) {
            case 'top':
                top = rect.top - tipRect.height - 8;
                left = rect.left + (rect.width - tipRect.width) / 2;
                break;
            case 'bottom':
                top = rect.bottom + 8;
                left = rect.left + (rect.width - tipRect.width) / 2;
                break;
            case 'left':
                top = rect.top + (rect.height - tipRect.height) / 2;
                left = rect.left - tipRect.width - 8;
                break;
            case 'right':
                top = rect.top + (rect.height - tipRect.height) / 2;
                left = rect.right + 8;
                break;
        }

        this.tooltip.style.top = `${top}px`;
        this.tooltip.style.left = `${left}px`;
    }

    _hideTooltip() {
        if (this.tooltip) {
            this.tooltip.remove();
            this.tooltip = null;
        }
    }
}
```

---

### Dropdown Hook (hooks/dropdown.js)

```javascript
/**
 * Dropdown hook for click-outside-to-close behavior
 */

export class DropdownHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        this.closeOnEscape = config.closeOnEscape !== false;
        this.closeOnClickOutside = config.closeOnClickOutside !== false;

        this._bindEvents();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
    }

    destroyed(el) {
        this._unbindEvents();
    }

    _bindEvents() {
        this._onClickOutside = this._handleClickOutside.bind(this);
        this._onKeyDown = this._handleKeyDown.bind(this);

        // Delay to avoid immediate trigger
        setTimeout(() => {
            document.addEventListener('click', this._onClickOutside);
            document.addEventListener('keydown', this._onKeyDown);
        }, 0);
    }

    _unbindEvents() {
        document.removeEventListener('click', this._onClickOutside);
        document.removeEventListener('keydown', this._onKeyDown);
    }

    _handleClickOutside(e) {
        if (!this.closeOnClickOutside) return;

        if (!this.el.contains(e.target)) {
            this.pushEvent('close', {});
        }
    }

    _handleKeyDown(e) {
        if (!this.closeOnEscape) return;

        if (e.key === 'Escape') {
            e.preventDefault();
            this.pushEvent('close', {});
        }
    }
}
```

---

### FocusTrap Hook (hooks/focustrap.js)

> **VangoUI Amendment**: Required for accessible Modal/Dialog components.

```javascript
/**
 * FocusTrap hook for modal accessibility
 * 
 * Constrains keyboard navigation (Tab/Shift+Tab) to a specific container.
 * Critical for Modal/Dialog accessibility compliance (WCAG 2.1).
 */

export class FocusTrapHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.pushEvent = pushEvent;
        this.active = config.active !== false;
        
        // Save previously focused element to restore later
        this.restoreFocusTo = document.activeElement;

        this._onKeyDown = this._handleKeyDown.bind(this);
        this.el.addEventListener('keydown', this._onKeyDown);

        if (this.active) {
            this._focusFirst();
        }
    }

    updated(el, config, pushEvent) {
        this.active = config.active !== false;
        this.pushEvent = pushEvent;
    }

    destroyed(el) {
        this.el.removeEventListener('keydown', this._onKeyDown);
        
        // Restore focus on close
        if (this.restoreFocusTo && typeof this.restoreFocusTo.focus === 'function') {
            this.restoreFocusTo.focus();
        }
    }

    _handleKeyDown(e) {
        if (!this.active || e.key !== 'Tab') return;

        const focusable = this._getFocusableElements();
        
        if (focusable.length === 0) {
            e.preventDefault();
            return;
        }

        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (e.shiftKey) { // Shift + Tab
            if (document.activeElement === first) {
                e.preventDefault();
                last.focus();
            }
        } else { // Tab
            if (document.activeElement === last) {
                e.preventDefault();
                first.focus();
            }
        }
    }

    _focusFirst() {
        const focusable = this._getFocusableElements();
        if (focusable.length > 0) {
            focusable[0].focus();
        } else {
            // Fallback: focus container itself with tabindex
            this.el.setAttribute('tabindex', '-1');
            this.el.focus();
        }
    }

    _getFocusableElements() {
        return this.el.querySelectorAll(
            'button:not([disabled]), ' +
            '[href], ' +
            'input:not([disabled]), ' +
            'select:not([disabled]), ' +
            'textarea:not([disabled]), ' +
            '[tabindex]:not([tabindex="-1"])'
        );
    }
}
```

---

### Portal Hook (hooks/portal.js)

> **VangoUI Amendment**: Required for proper z-index stacking of Modal/Dialog/Tooltip/Popover components.

```javascript
/**
 * Portal hook for DOM element repositioning
 * 
 * Moves the DOM element to the end of document.body to escape
 * overflow: hidden containers and ensure correct z-index stacking context.
 * Used by Dialogs, Tooltips, Popovers, and other overlay components.
 */

export class PortalHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.pushEvent = pushEvent;
        this.target = config.target || 'body';
        
        // Create placeholder to keep VNode tree structure aligned
        this.placeholder = document.createComment('portal-placeholder');
        
        // Insert placeholder before moving
        if (el.parentNode) {
            el.parentNode.insertBefore(this.placeholder, el);
        }
        
        // Get or create portal container
        let portalRoot = document.getElementById('vango-portal-root');
        if (!portalRoot) {
            portalRoot = document.createElement('div');
            portalRoot.id = 'vango-portal-root';
            portalRoot.style.cssText = 'position: relative; z-index: 9999;';
            document.body.appendChild(portalRoot);
        }

        // Move element to portal
        portalRoot.appendChild(el);
    }

    updated(el, config, pushEvent) {
        this.pushEvent = pushEvent;
    }

    destroyed(el) {
        // Move back before destruction (optional, but good for cleanup)
        if (this.placeholder && this.placeholder.parentNode) {
            this.placeholder.parentNode.insertBefore(this.el, this.placeholder);
            this.placeholder.remove();
        }
    }
}
```

---

## Build Process

### package.json

```json
{
  "name": "@vango/client",
  "version": "2.0.0",
  "description": "Vango thin client",
  "main": "dist/vango.min.js",
  "scripts": {
    "build": "node build.js",
    "build:dev": "node build.js --dev",
    "watch": "node build.js --watch"
  },
  "devDependencies": {
    "esbuild": "^0.19.0"
  }
}
```

### build.js

```javascript
const esbuild = require('esbuild');
const fs = require('fs');

const isDev = process.argv.includes('--dev');
const isWatch = process.argv.includes('--watch');

const options = {
    entryPoints: ['src/index.js'],
    bundle: true,
    outfile: isDev ? 'dist/vango.js' : 'dist/vango.min.js',
    minify: !isDev,
    sourcemap: isDev,
    target: ['es2018'],
    format: 'iife',
    globalName: 'Vango'
};

if (isWatch) {
    esbuild.context(options).then(ctx => {
        ctx.watch();
        console.log('Watching...');
    });
} else {
    esbuild.build(options).then(result => {
        const stat = fs.statSync(options.outfile);
        console.log(`Built ${options.outfile} (${(stat.size / 1024).toFixed(2)} KB)`);
    });
}
```

---

## Testing Strategy

### Unit Tests

```javascript
// test/codec.test.js
import { BinaryCodec, EventType, PatchType } from '../src/codec.js';

describe('BinaryCodec', () => {
    const codec = new BinaryCodec();

    describe('varint', () => {
        it('encodes small numbers in one byte', () => {
            const encoded = codec.encodeVarint(42);
            expect(encoded.length).toBe(1);
            expect(encoded[0]).toBe(42);
        });

        it('encodes larger numbers in multiple bytes', () => {
            const encoded = codec.encodeVarint(300);
            expect(encoded.length).toBe(2);
        });

        it('round-trips correctly', () => {
            for (const value of [0, 1, 127, 128, 16383, 16384, 100000]) {
                const encoded = codec.encodeVarint(value);
                const { value: decoded } = codec.decodeVarint(encoded, 0);
                expect(decoded).toBe(value);
            }
        });
    });

    describe('events', () => {
        it('encodes click event', () => {
            const buffer = codec.encodeEvent(EventType.CLICK, 'h42', null);
            expect(buffer[0]).toBe(0x00); // Frame type
            expect(buffer[1]).toBe(EventType.CLICK);
            expect(buffer[2]).toBe(42); // HID
        });

        it('encodes input event with value', () => {
            const buffer = codec.encodeEvent(EventType.INPUT, 'h1', { value: 'hello' });
            expect(buffer.length).toBeGreaterThan(3);
        });
    });

    describe('patches', () => {
        it('decodes SET_TEXT patch', () => {
            // [count=1][type][hid][len][text]
            const buffer = new Uint8Array([1, PatchType.SET_TEXT, 5, 5, 104, 101, 108, 108, 111]);
            const patches = codec.decodePatches(buffer);

            expect(patches.length).toBe(1);
            expect(patches[0].type).toBe(PatchType.SET_TEXT);
            expect(patches[0].hid).toBe('h5');
            expect(patches[0].text).toBe('hello');
        });
    });
});
```

### Integration Tests

```javascript
// test/integration.test.js
import { VangoClient } from '../src/index.js';

describe('VangoClient', () => {
    let client;
    let mockWs;

    beforeEach(() => {
        // Mock WebSocket
        mockWs = {
            send: jest.fn(),
            close: jest.fn(),
            readyState: WebSocket.OPEN
        };
        global.WebSocket = jest.fn(() => mockWs);

        // Mock DOM
        document.body.innerHTML = `
            <div data-hid="h1">
                <button data-hid="h2" data-on-click="true">Click</button>
            </div>
        `;

        client = new VangoClient({ reconnect: false });
    });

    afterEach(() => {
        client.destroy();
    });

    it('builds node map on init', () => {
        expect(client.nodeMap.size).toBe(2);
        expect(client.nodeMap.get('h1')).toBe(document.querySelector('[data-hid="h1"]'));
    });

    it('sends click event', () => {
        const button = document.querySelector('button');
        button.click();

        expect(mockWs.send).toHaveBeenCalled();
    });

    it('applies SET_TEXT patch', () => {
        const patch = { type: 0x01, hid: 'h1', text: 'Updated' };
        client.patchApplier.applyPatch(patch);

        expect(document.querySelector('[data-hid="h1"]').textContent).toBe('Updated');
    });
});
```

### Browser Tests

```javascript
// test/browser.test.js (run with Playwright)
const { test, expect } = require('@playwright/test');

test('client connects and handles events', async ({ page }) => {
    await page.goto('/test-page');

    // Wait for WebSocket connection
    await page.waitForFunction(() => window.__vango__?.connected);

    // Click button
    await page.click('[data-hid="h1"]');

    // Verify event was sent (check mock server)
    // ...
});

test('client applies patches', async ({ page }) => {
    await page.goto('/test-page');
    await page.waitForFunction(() => window.__vango__?.connected);

    // Inject patch (simulate server message)
    await page.evaluate(() => {
        const patch = { type: 0x01, hid: 'h1', text: 'Server Updated' };
        window.__vango__.patchApplier.applyPatch(patch);
    });

    // Verify DOM updated
    const text = await page.textContent('[data-hid="h1"]');
    expect(text).toBe('Server Updated');
});
```

---

## Size Analysis

Expected bundle size breakdown:

| Module | Raw | Minified | Gzipped |
|--------|-----|----------|---------|
| index.js | 8 KB | 4 KB | 1.8 KB |
| codec.js | 6 KB | 3 KB | 1.5 KB |
| events.js | 5 KB | 2.5 KB | 1.2 KB |
| patches.js | 4 KB | 2 KB | 1.0 KB |
| optimistic.js | 2 KB | 1 KB | 0.5 KB |
| hooks/manager.js | 2 KB | 1 KB | 0.5 KB |
| hooks/sortable.js | 3 KB | 1.5 KB | 0.8 KB |
| hooks/tooltip.js | 1.5 KB | 0.8 KB | 0.4 KB |
| hooks/dropdown.js | 1 KB | 0.5 KB | 0.3 KB |
| **Total** | **32.5 KB** | **16.3 KB** | **8 KB** |

With tree-shaking (if hooks not used): **~6 KB gzipped**

---

## Browser Compatibility

| Browser | Version | Support |
|---------|---------|---------|
| Chrome | 60+ | Full |
| Firefox | 55+ | Full |
| Safari | 11+ | Full |
| Edge | 79+ | Full |
| IE | - | Not supported |

Required features:
- WebSocket (all modern browsers)
- ArrayBuffer (all modern browsers)
- TextEncoder/TextDecoder (polyfill for Safari 10.1)
- classList, dataset (all modern browsers)

---

## Exit Criteria

Phase 5 is complete when:

1. [x] Core client implementation complete
2. [x] Binary codec matches server protocol exactly
3. [x] All event types captured and encoded
4. [x] All patch types applied correctly
5. [x] WebSocket connection with auto-reconnect
6. [x] Optimistic updates working
7. [x] **All 7 standard hooks from Spec §8.4 implemented** (Sortable, Draggable, Droppable, Resizable, Tooltip, Dropdown, Collapsible)
8. [x] **VangoUI helper hooks implemented** (FocusTrap, Portal, Dialog, Popover)
9. [x] Bundle size ~15KB gzipped (actual: **15.90 KB** with all 11 hooks)
10. [x] Unit tests passing (30 codec tests)
11. [x] Integration tests passing (18 tests)
12. [x] **Total: 48 tests passing**
13. [ ] Browser tests passing (requires Phase 6 SSR for full e2e)
14. [x] Documentation complete

---

## Dependencies

- **Requires**: Phase 3 (Protocol specification)
- **Required by**: Phase 6 (SSR injects client script)

---

*Phase 5 Specification - Version 1.0*
