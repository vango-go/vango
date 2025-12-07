/**
 * Binary Codec for Vango Protocol
 *
 * Handles encoding/decoding of events and patches using the binary protocol
 * that matches the Go server implementation in pkg/protocol/.
 */

import { hidToInt, intToHid, concat } from './utils.js';

/**
 * Event type constants - must match pkg/protocol/event.go
 */
export const EventType = {
    // Mouse events (0x01-0x07)
    CLICK: 0x01,
    DBLCLICK: 0x02,
    MOUSEDOWN: 0x03,
    MOUSEUP: 0x04,
    MOUSEMOVE: 0x05,
    MOUSEENTER: 0x06,
    MOUSELEAVE: 0x07,

    // Form events (0x10-0x14)
    INPUT: 0x10,
    CHANGE: 0x11,
    SUBMIT: 0x12,
    FOCUS: 0x13,
    BLUR: 0x14,

    // Keyboard events (0x20-0x22)
    KEYDOWN: 0x20,
    KEYUP: 0x21,
    KEYPRESS: 0x22,

    // Scroll/Resize events (0x30-0x31)
    SCROLL: 0x30,
    RESIZE: 0x31,

    // Touch events (0x40-0x42)
    TOUCHSTART: 0x40,
    TOUCHMOVE: 0x41,
    TOUCHEND: 0x42,

    // Drag events (0x50-0x52)
    DRAGSTART: 0x50,
    DRAGEND: 0x51,
    DROP: 0x52,

    // Special events (0x60+)
    HOOK: 0x60,
    NAVIGATE: 0x70,
    CUSTOM: 0xFF,
};

/**
 * Patch type constants - must match pkg/protocol/patch.go
 */
export const PatchType = {
    SET_TEXT: 0x01,
    SET_ATTR: 0x02,
    REMOVE_ATTR: 0x03,
    INSERT_NODE: 0x04,
    REMOVE_NODE: 0x05,
    MOVE_NODE: 0x06,
    REPLACE_NODE: 0x07,
    SET_VALUE: 0x08,
    SET_CHECKED: 0x09,
    SET_SELECTED: 0x0A,
    FOCUS: 0x0B,
    BLUR: 0x0C,
    SCROLL_TO: 0x0D,
    ADD_CLASS: 0x10,
    REMOVE_CLASS: 0x11,
    TOGGLE_CLASS: 0x12,
    SET_STYLE: 0x13,
    REMOVE_STYLE: 0x14,
    SET_DATA: 0x15,
    DISPATCH: 0x20,
    EVAL: 0x21,
};

/**
 * Key modifier flags - must match pkg/protocol/event.go
 */
export const KeyMod = {
    CTRL: 0x01,
    SHIFT: 0x02,
    ALT: 0x04,
    META: 0x08,
};

/**
 * VNode type constants for wire format
 */
const VNodeType = {
    ELEMENT: 0x01,
    TEXT: 0x02,
    FRAGMENT: 0x03,
};

/**
 * Hook value type constants
 */
const HookValueType = {
    NULL: 0x00,
    BOOL: 0x01,
    INT: 0x02,
    FLOAT: 0x03,
    STRING: 0x04,
    ARRAY: 0x05,
    OBJECT: 0x06,
};

/**
 * Binary codec for Vango protocol
 */
export class BinaryCodec {
    constructor() {
        this.textEncoder = new TextEncoder();
        this.textDecoder = new TextDecoder();
    }

    /**
     * Encode event to binary buffer
     * Format: [seq][type][hid][payload...]
     */
    encodeEvent(seq, type, hid, data = null) {
        const parts = [];

        // Sequence number
        parts.push(this.encodeUvarint(seq));

        // Event type
        parts.push(new Uint8Array([type]));

        // HID as string
        parts.push(this.encodeString(hid));

        // Payload (depends on event type)
        switch (type) {
            case EventType.INPUT:
            case EventType.CHANGE:
                // String value
                parts.push(this.encodeString(data?.value || ''));
                break;

            case EventType.SUBMIT:
                // Form fields map
                this.encodeFormData(parts, data);
                break;

            case EventType.KEYDOWN:
            case EventType.KEYUP:
            case EventType.KEYPRESS:
                this.encodeKeyboardEvent(parts, data);
                break;

            case EventType.MOUSEDOWN:
            case EventType.MOUSEUP:
            case EventType.MOUSEMOVE:
                this.encodeMouseEvent(parts, data);
                break;

            case EventType.SCROLL:
                this.encodeScrollEvent(parts, data);
                break;

            case EventType.RESIZE:
                this.encodeResizeEvent(parts, data);
                break;

            case EventType.TOUCHSTART:
            case EventType.TOUCHMOVE:
            case EventType.TOUCHEND:
                this.encodeTouchEvent(parts, data);
                break;

            case EventType.DRAGSTART:
            case EventType.DRAGEND:
            case EventType.DROP:
                this.encodeDragEvent(parts, data);
                break;

            case EventType.HOOK:
                this.encodeHookEvent(parts, data);
                break;

            case EventType.NAVIGATE:
                parts.push(this.encodeString(data?.path || ''));
                parts.push(new Uint8Array([data?.replace ? 1 : 0]));
                break;

            case EventType.CUSTOM:
                parts.push(this.encodeString(data?.name || ''));
                parts.push(this.encodeLenBytes(data?.data || new Uint8Array(0)));
                break;

            // No payload for simple events
            case EventType.CLICK:
            case EventType.DBLCLICK:
            case EventType.FOCUS:
            case EventType.BLUR:
            case EventType.MOUSEENTER:
            case EventType.MOUSELEAVE:
            default:
                // No additional data
                break;
        }

        return concat(parts);
    }

    /**
     * Decode patches from binary buffer
     * Format: [seq][count][patch...]
     */
    decodePatches(buffer) {
        let offset = 0;

        // Read sequence number
        const { value: seq, bytesRead: seqBytes } = this.decodeUvarint(buffer, offset);
        offset += seqBytes;

        // Read patch count
        const { value: count, bytesRead: countBytes } = this.decodeUvarint(buffer, offset);
        offset += countBytes;

        const patches = [];
        for (let i = 0; i < count; i++) {
            const { patch, bytesRead } = this.decodePatch(buffer, offset);
            patches.push(patch);
            offset += bytesRead;
        }

        return { seq, patches };
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
        const { value: hid, bytesRead: hidBytes } = this.decodeString(buffer, offset);
        patch.hid = hid;
        offset += hidBytes;

        // Payload (depends on patch type)
        switch (patch.type) {
            case PatchType.SET_TEXT:
            case PatchType.SET_VALUE: {
                const { value, bytesRead } = this.decodeString(buffer, offset);
                patch.value = value;
                offset += bytesRead;
                break;
            }

            case PatchType.SET_ATTR:
            case PatchType.SET_STYLE:
            case PatchType.SET_DATA: {
                const { value: key, bytesRead: keyBytes } = this.decodeString(buffer, offset);
                offset += keyBytes;
                const { value: val, bytesRead: valBytes } = this.decodeString(buffer, offset);
                offset += valBytes;
                patch.key = key;
                patch.value = val;
                break;
            }

            case PatchType.REMOVE_ATTR:
            case PatchType.REMOVE_STYLE: {
                const { value, bytesRead } = this.decodeString(buffer, offset);
                patch.key = value;
                offset += bytesRead;
                break;
            }

            case PatchType.ADD_CLASS:
            case PatchType.REMOVE_CLASS:
            case PatchType.TOGGLE_CLASS: {
                const { value, bytesRead } = this.decodeString(buffer, offset);
                patch.className = value;
                offset += bytesRead;
                break;
            }

            case PatchType.INSERT_NODE: {
                const { value: parentID, bytesRead: parentBytes } = this.decodeString(buffer, offset);
                offset += parentBytes;
                const { value: index, bytesRead: indexBytes } = this.decodeUvarint(buffer, offset);
                offset += indexBytes;
                const { vnode, bytesRead: vnodeBytes } = this.decodeVNode(buffer, offset);
                offset += vnodeBytes;
                patch.parentID = parentID;
                patch.index = index;
                patch.vnode = vnode;
                break;
            }

            case PatchType.REMOVE_NODE:
                // No additional data (HID is sufficient)
                break;

            case PatchType.MOVE_NODE: {
                const { value: parentID, bytesRead: parentBytes } = this.decodeString(buffer, offset);
                offset += parentBytes;
                const { value: index, bytesRead: indexBytes } = this.decodeUvarint(buffer, offset);
                offset += indexBytes;
                patch.parentID = parentID;
                patch.index = index;
                break;
            }

            case PatchType.REPLACE_NODE: {
                const { vnode, bytesRead: vnodeBytes } = this.decodeVNode(buffer, offset);
                offset += vnodeBytes;
                patch.vnode = vnode;
                break;
            }

            case PatchType.SET_CHECKED:
            case PatchType.SET_SELECTED: {
                patch.value = buffer[offset++] === 1;
                break;
            }

            case PatchType.FOCUS:
            case PatchType.BLUR:
                // No payload
                break;

            case PatchType.SCROLL_TO: {
                const { value: x, bytesRead: xBytes } = this.decodeSvarint(buffer, offset);
                offset += xBytes;
                const { value: y, bytesRead: yBytes } = this.decodeSvarint(buffer, offset);
                offset += yBytes;
                patch.x = x;
                patch.y = y;
                patch.behavior = buffer[offset++]; // 0 = instant, 1 = smooth
                break;
            }

            case PatchType.DISPATCH: {
                const { value: eventName, bytesRead: nameBytes } = this.decodeString(buffer, offset);
                offset += nameBytes;
                const { value: detail, bytesRead: detailBytes } = this.decodeString(buffer, offset);
                offset += detailBytes;
                patch.eventName = eventName;
                patch.detail = detail;
                break;
            }

            case PatchType.EVAL: {
                const { value, bytesRead } = this.decodeString(buffer, offset);
                patch.code = value;
                offset += bytesRead;
                break;
            }

            default:
                // Unknown patch type - skip for forward compatibility
                break;
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
            case VNodeType.ELEMENT: {
                vnode.type = 'element';

                // Tag name
                const { value: tag, bytesRead: tagBytes } = this.decodeString(buffer, offset);
                vnode.tag = tag;
                offset += tagBytes;

                // HID (empty string = no hid)
                const { value: hid, bytesRead: hidBytes } = this.decodeString(buffer, offset);
                vnode.hid = hid || null;
                offset += hidBytes;

                // Attributes count and key-value pairs
                const { value: attrCount, bytesRead: attrCountBytes } = this.decodeUvarint(buffer, offset);
                offset += attrCountBytes;
                vnode.attrs = {};

                for (let i = 0; i < attrCount; i++) {
                    const { value: key, bytesRead: keyBytes } = this.decodeString(buffer, offset);
                    offset += keyBytes;
                    const { value: val, bytesRead: valBytes } = this.decodeString(buffer, offset);
                    offset += valBytes;
                    vnode.attrs[key] = val;
                }

                // Children count and nodes
                const { value: childCount, bytesRead: childCountBytes } = this.decodeUvarint(buffer, offset);
                offset += childCountBytes;
                vnode.children = [];

                for (let i = 0; i < childCount; i++) {
                    const { vnode: child, bytesRead: childBytes } = this.decodeVNode(buffer, offset);
                    vnode.children.push(child);
                    offset += childBytes;
                }
                break;
            }

            case VNodeType.TEXT: {
                vnode.type = 'text';
                const { value: text, bytesRead: textBytes } = this.decodeString(buffer, offset);
                vnode.text = text;
                offset += textBytes;
                break;
            }

            case VNodeType.FRAGMENT: {
                vnode.type = 'fragment';
                const { value: childCount, bytesRead: countBytes } = this.decodeUvarint(buffer, offset);
                offset += countBytes;
                vnode.children = [];

                for (let i = 0; i < childCount; i++) {
                    const { vnode: child, bytesRead: childBytes } = this.decodeVNode(buffer, offset);
                    vnode.children.push(child);
                    offset += childBytes;
                }
                break;
            }

            default:
                vnode.type = 'unknown';
        }

        return { vnode, bytesRead: offset - startOffset };
    }

    /**
     * Encode string with length prefix (varint length + UTF-8 bytes)
     */
    encodeString(str) {
        const bytes = this.textEncoder.encode(str);
        const length = this.encodeUvarint(bytes.length);
        return concat([length, bytes]);
    }

    /**
     * Decode string with length prefix
     */
    decodeString(buffer, offset) {
        const { value: length, bytesRead: lengthBytes } = this.decodeUvarint(buffer, offset);
        const strBytes = buffer.slice(offset + lengthBytes, offset + lengthBytes + length);
        const value = this.textDecoder.decode(strBytes);
        return { value, bytesRead: lengthBytes + length };
    }

    /**
     * Encode unsigned varint (protobuf-style)
     */
    encodeUvarint(value) {
        const bytes = [];
        while (value > 0x7F) {
            bytes.push((value & 0x7F) | 0x80);
            value >>>= 7;
        }
        bytes.push(value & 0x7F);
        return new Uint8Array(bytes);
    }

    /**
     * Decode unsigned varint
     */
    decodeUvarint(buffer, offset) {
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
     * Encode signed varint using ZigZag encoding
     */
    encodeSvarint(value) {
        // ZigZag encode: (n << 1) ^ (n >> 63)
        const zigzag = (value << 1) ^ (value >> 31);
        return this.encodeUvarint(zigzag >>> 0); // Convert to unsigned
    }

    /**
     * Decode signed varint using ZigZag encoding
     */
    decodeSvarint(buffer, offset) {
        const { value: zigzag, bytesRead } = this.decodeUvarint(buffer, offset);
        // ZigZag decode: (n >>> 1) ^ -(n & 1)
        const value = (zigzag >>> 1) ^ -(zigzag & 1);
        return { value, bytesRead };
    }

    /**
     * Encode length-prefixed bytes
     */
    encodeLenBytes(bytes) {
        const length = this.encodeUvarint(bytes.length);
        return concat([length, bytes]);
    }

    /**
     * Decode length-prefixed bytes
     */
    decodeLenBytes(buffer, offset) {
        const { value: length, bytesRead: lengthBytes } = this.decodeUvarint(buffer, offset);
        const bytes = buffer.slice(offset + lengthBytes, offset + lengthBytes + length);
        return { value: bytes, bytesRead: lengthBytes + length };
    }

    /**
     * Encode form data map
     */
    encodeFormData(parts, formData) {
        if (!formData) {
            parts.push(this.encodeUvarint(0));
            return;
        }

        const entries = [];
        if (formData instanceof FormData) {
            for (const [key, value] of formData.entries()) {
                entries.push({ key, value: String(value) });
            }
        } else if (typeof formData === 'object') {
            for (const [key, value] of Object.entries(formData)) {
                entries.push({ key, value: String(value) });
            }
        }

        parts.push(this.encodeUvarint(entries.length));
        for (const { key, value } of entries) {
            parts.push(this.encodeString(key));
            parts.push(this.encodeString(value));
        }
    }

    /**
     * Encode keyboard event
     */
    encodeKeyboardEvent(parts, data) {
        parts.push(this.encodeString(data?.key || ''));

        let modifiers = 0;
        if (data?.ctrlKey) modifiers |= KeyMod.CTRL;
        if (data?.shiftKey) modifiers |= KeyMod.SHIFT;
        if (data?.altKey) modifiers |= KeyMod.ALT;
        if (data?.metaKey) modifiers |= KeyMod.META;
        parts.push(new Uint8Array([modifiers]));
    }

    /**
     * Encode mouse event
     */
    encodeMouseEvent(parts, data) {
        parts.push(this.encodeSvarint(data?.clientX || 0));
        parts.push(this.encodeSvarint(data?.clientY || 0));
        parts.push(new Uint8Array([data?.button || 0]));

        let modifiers = 0;
        if (data?.ctrlKey) modifiers |= KeyMod.CTRL;
        if (data?.shiftKey) modifiers |= KeyMod.SHIFT;
        if (data?.altKey) modifiers |= KeyMod.ALT;
        if (data?.metaKey) modifiers |= KeyMod.META;
        parts.push(new Uint8Array([modifiers]));
    }

    /**
     * Encode scroll event
     */
    encodeScrollEvent(parts, data) {
        parts.push(this.encodeSvarint(data?.scrollTop || 0));
        parts.push(this.encodeSvarint(data?.scrollLeft || 0));
    }

    /**
     * Encode resize event
     */
    encodeResizeEvent(parts, data) {
        parts.push(this.encodeSvarint(data?.width || 0));
        parts.push(this.encodeSvarint(data?.height || 0));
    }

    /**
     * Encode touch event
     */
    encodeTouchEvent(parts, data) {
        const touches = data?.touches || [];
        parts.push(this.encodeUvarint(touches.length));
        for (const touch of touches) {
            parts.push(this.encodeSvarint(touch.id || 0));
            parts.push(this.encodeSvarint(touch.clientX || 0));
            parts.push(this.encodeSvarint(touch.clientY || 0));
        }
    }

    /**
     * Encode drag event
     */
    encodeDragEvent(parts, data) {
        parts.push(this.encodeSvarint(data?.clientX || 0));
        parts.push(this.encodeSvarint(data?.clientY || 0));

        let modifiers = 0;
        if (data?.ctrlKey) modifiers |= KeyMod.CTRL;
        if (data?.shiftKey) modifiers |= KeyMod.SHIFT;
        if (data?.altKey) modifiers |= KeyMod.ALT;
        if (data?.metaKey) modifiers |= KeyMod.META;
        parts.push(new Uint8Array([modifiers]));
    }

    /**
     * Encode hook event
     */
    encodeHookEvent(parts, data) {
        parts.push(this.encodeString(data?.name || ''));
        this.encodeHookData(parts, data?.data || {});
    }

    /**
     * Encode hook data map
     */
    encodeHookData(parts, data) {
        const entries = Object.entries(data);
        parts.push(this.encodeUvarint(entries.length));
        for (const [key, value] of entries) {
            parts.push(this.encodeString(key));
            this.encodeHookValue(parts, value);
        }
    }

    /**
     * Encode single hook value
     */
    encodeHookValue(parts, value) {
        if (value === null || value === undefined) {
            parts.push(new Uint8Array([HookValueType.NULL]));
        } else if (typeof value === 'boolean') {
            parts.push(new Uint8Array([HookValueType.BOOL, value ? 1 : 0]));
        } else if (typeof value === 'number' && Number.isInteger(value)) {
            parts.push(new Uint8Array([HookValueType.INT]));
            parts.push(this.encodeSvarint(value));
        } else if (typeof value === 'number') {
            parts.push(new Uint8Array([HookValueType.FLOAT]));
            parts.push(this.encodeFloat64(value));
        } else if (typeof value === 'string') {
            parts.push(new Uint8Array([HookValueType.STRING]));
            parts.push(this.encodeString(value));
        } else if (Array.isArray(value)) {
            parts.push(new Uint8Array([HookValueType.ARRAY]));
            parts.push(this.encodeUvarint(value.length));
            for (const item of value) {
                this.encodeHookValue(parts, item);
            }
        } else if (typeof value === 'object') {
            parts.push(new Uint8Array([HookValueType.OBJECT]));
            const entries = Object.entries(value);
            parts.push(this.encodeUvarint(entries.length));
            for (const [k, v] of entries) {
                parts.push(this.encodeString(k));
                this.encodeHookValue(parts, v);
            }
        } else {
            // Unknown type, encode as null
            parts.push(new Uint8Array([HookValueType.NULL]));
        }
    }

    /**
     * Encode 64-bit float (little-endian)
     */
    encodeFloat64(value) {
        const buffer = new ArrayBuffer(8);
        new DataView(buffer).setFloat64(0, value, true);
        return new Uint8Array(buffer);
    }
}
