var Vango = (() => {
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __name = (target, value) => __defProp(target, "name", { value, configurable: true });
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

  // src/index.js
  var src_exports = {};
  __export(src_exports, {
    EventType: () => EventType,
    VangoClient: () => VangoClient,
    default: () => src_default
  });

  // src/utils.js
  function concat(arrays) {
    let totalLength = 0;
    for (let i = 0; i < arrays.length; i++) {
      totalLength += arrays[i].length;
    }
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (let i = 0; i < arrays.length; i++) {
      result.set(arrays[i], offset);
      offset += arrays[i].length;
    }
    return result;
  }
  __name(concat, "concat");

  // src/codec.js
  var EventType = {
    // Mouse events (0x01-0x07)
    CLICK: 1,
    DBLCLICK: 2,
    MOUSEDOWN: 3,
    MOUSEUP: 4,
    MOUSEMOVE: 5,
    MOUSEENTER: 6,
    MOUSELEAVE: 7,
    // Form events (0x10-0x14)
    INPUT: 16,
    CHANGE: 17,
    SUBMIT: 18,
    FOCUS: 19,
    BLUR: 20,
    // Keyboard events (0x20-0x22)
    KEYDOWN: 32,
    KEYUP: 33,
    KEYPRESS: 34,
    // Scroll/Resize events (0x30-0x31)
    SCROLL: 48,
    RESIZE: 49,
    // Touch events (0x40-0x42)
    TOUCHSTART: 64,
    TOUCHMOVE: 65,
    TOUCHEND: 66,
    // Drag events (0x50-0x52)
    DRAGSTART: 80,
    DRAGEND: 81,
    DROP: 82,
    // Special events (0x60+)
    HOOK: 96,
    NAVIGATE: 112,
    CUSTOM: 255
  };
  var PatchType = {
    SET_TEXT: 1,
    SET_ATTR: 2,
    REMOVE_ATTR: 3,
    INSERT_NODE: 4,
    REMOVE_NODE: 5,
    MOVE_NODE: 6,
    REPLACE_NODE: 7,
    SET_VALUE: 8,
    SET_CHECKED: 9,
    SET_SELECTED: 10,
    FOCUS: 11,
    BLUR: 12,
    SCROLL_TO: 13,
    ADD_CLASS: 16,
    REMOVE_CLASS: 17,
    TOGGLE_CLASS: 18,
    SET_STYLE: 19,
    REMOVE_STYLE: 20,
    SET_DATA: 21,
    DISPATCH: 32,
    EVAL: 33
  };
  var KeyMod = {
    CTRL: 1,
    SHIFT: 2,
    ALT: 4,
    META: 8
  };
  var VNodeType = {
    ELEMENT: 1,
    TEXT: 2,
    FRAGMENT: 3
  };
  var HookValueType = {
    NULL: 0,
    BOOL: 1,
    INT: 2,
    FLOAT: 3,
    STRING: 4,
    ARRAY: 5,
    OBJECT: 6
  };
  var _BinaryCodec = class _BinaryCodec {
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
      parts.push(this.encodeUvarint(seq));
      parts.push(new Uint8Array([type]));
      parts.push(this.encodeString(hid));
      switch (type) {
        case EventType.INPUT:
        case EventType.CHANGE:
          parts.push(this.encodeString((data == null ? void 0 : data.value) || ""));
          break;
        case EventType.SUBMIT:
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
          parts.push(this.encodeString((data == null ? void 0 : data.path) || ""));
          parts.push(new Uint8Array([(data == null ? void 0 : data.replace) ? 1 : 0]));
          break;
        case EventType.CUSTOM:
          parts.push(this.encodeString((data == null ? void 0 : data.name) || ""));
          parts.push(this.encodeLenBytes((data == null ? void 0 : data.data) || new Uint8Array(0)));
          break;
        case EventType.CLICK:
        case EventType.DBLCLICK:
        case EventType.FOCUS:
        case EventType.BLUR:
        case EventType.MOUSEENTER:
        case EventType.MOUSELEAVE:
        default:
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
      const { value: seq, bytesRead: seqBytes } = this.decodeUvarint(buffer, offset);
      offset += seqBytes;
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
      patch.type = buffer[offset++];
      const { value: hid, bytesRead: hidBytes } = this.decodeString(buffer, offset);
      patch.hid = hid;
      offset += hidBytes;
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
          break;
        case PatchType.SCROLL_TO: {
          const { value: x, bytesRead: xBytes } = this.decodeSvarint(buffer, offset);
          offset += xBytes;
          const { value: y, bytesRead: yBytes } = this.decodeSvarint(buffer, offset);
          offset += yBytes;
          patch.x = x;
          patch.y = y;
          patch.behavior = buffer[offset++];
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
      const nodeType = buffer[offset++];
      switch (nodeType) {
        case VNodeType.ELEMENT: {
          vnode.type = "element";
          const { value: tag, bytesRead: tagBytes } = this.decodeString(buffer, offset);
          vnode.tag = tag;
          offset += tagBytes;
          const { value: hid, bytesRead: hidBytes } = this.decodeString(buffer, offset);
          vnode.hid = hid || null;
          offset += hidBytes;
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
          vnode.type = "text";
          const { value: text, bytesRead: textBytes } = this.decodeString(buffer, offset);
          vnode.text = text;
          offset += textBytes;
          break;
        }
        case VNodeType.FRAGMENT: {
          vnode.type = "fragment";
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
          vnode.type = "unknown";
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
      while (value > 127) {
        bytes.push(value & 127 | 128);
        value >>>= 7;
      }
      bytes.push(value & 127);
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
        value |= (byte & 127) << shift;
        if ((byte & 128) === 0) {
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
      const zigzag = value << 1 ^ value >> 31;
      return this.encodeUvarint(zigzag >>> 0);
    }
    /**
     * Decode signed varint using ZigZag encoding
     */
    decodeSvarint(buffer, offset) {
      const { value: zigzag, bytesRead } = this.decodeUvarint(buffer, offset);
      const value = zigzag >>> 1 ^ -(zigzag & 1);
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
      } else if (typeof formData === "object") {
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
      parts.push(this.encodeString((data == null ? void 0 : data.key) || ""));
      let modifiers = 0;
      if (data == null ? void 0 : data.ctrlKey)
        modifiers |= KeyMod.CTRL;
      if (data == null ? void 0 : data.shiftKey)
        modifiers |= KeyMod.SHIFT;
      if (data == null ? void 0 : data.altKey)
        modifiers |= KeyMod.ALT;
      if (data == null ? void 0 : data.metaKey)
        modifiers |= KeyMod.META;
      parts.push(new Uint8Array([modifiers]));
    }
    /**
     * Encode mouse event
     */
    encodeMouseEvent(parts, data) {
      parts.push(this.encodeSvarint((data == null ? void 0 : data.clientX) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.clientY) || 0));
      parts.push(new Uint8Array([(data == null ? void 0 : data.button) || 0]));
      let modifiers = 0;
      if (data == null ? void 0 : data.ctrlKey)
        modifiers |= KeyMod.CTRL;
      if (data == null ? void 0 : data.shiftKey)
        modifiers |= KeyMod.SHIFT;
      if (data == null ? void 0 : data.altKey)
        modifiers |= KeyMod.ALT;
      if (data == null ? void 0 : data.metaKey)
        modifiers |= KeyMod.META;
      parts.push(new Uint8Array([modifiers]));
    }
    /**
     * Encode scroll event
     */
    encodeScrollEvent(parts, data) {
      parts.push(this.encodeSvarint((data == null ? void 0 : data.scrollTop) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.scrollLeft) || 0));
    }
    /**
     * Encode resize event
     */
    encodeResizeEvent(parts, data) {
      parts.push(this.encodeSvarint((data == null ? void 0 : data.width) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.height) || 0));
    }
    /**
     * Encode touch event
     */
    encodeTouchEvent(parts, data) {
      const touches = (data == null ? void 0 : data.touches) || [];
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
      parts.push(this.encodeSvarint((data == null ? void 0 : data.clientX) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.clientY) || 0));
      let modifiers = 0;
      if (data == null ? void 0 : data.ctrlKey)
        modifiers |= KeyMod.CTRL;
      if (data == null ? void 0 : data.shiftKey)
        modifiers |= KeyMod.SHIFT;
      if (data == null ? void 0 : data.altKey)
        modifiers |= KeyMod.ALT;
      if (data == null ? void 0 : data.metaKey)
        modifiers |= KeyMod.META;
      parts.push(new Uint8Array([modifiers]));
    }
    /**
     * Encode hook event
     */
    encodeHookEvent(parts, data) {
      parts.push(this.encodeString((data == null ? void 0 : data.name) || ""));
      this.encodeHookData(parts, (data == null ? void 0 : data.data) || {});
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
      if (value === null || value === void 0) {
        parts.push(new Uint8Array([HookValueType.NULL]));
      } else if (typeof value === "boolean") {
        parts.push(new Uint8Array([HookValueType.BOOL, value ? 1 : 0]));
      } else if (typeof value === "number" && Number.isInteger(value)) {
        parts.push(new Uint8Array([HookValueType.INT]));
        parts.push(this.encodeSvarint(value));
      } else if (typeof value === "number") {
        parts.push(new Uint8Array([HookValueType.FLOAT]));
        parts.push(this.encodeFloat64(value));
      } else if (typeof value === "string") {
        parts.push(new Uint8Array([HookValueType.STRING]));
        parts.push(this.encodeString(value));
      } else if (Array.isArray(value)) {
        parts.push(new Uint8Array([HookValueType.ARRAY]));
        parts.push(this.encodeUvarint(value.length));
        for (const item of value) {
          this.encodeHookValue(parts, item);
        }
      } else if (typeof value === "object") {
        parts.push(new Uint8Array([HookValueType.OBJECT]));
        const entries = Object.entries(value);
        parts.push(this.encodeUvarint(entries.length));
        for (const [k, v] of entries) {
          parts.push(this.encodeString(k));
          this.encodeHookValue(parts, v);
        }
      } else {
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
  };
  __name(_BinaryCodec, "BinaryCodec");
  var BinaryCodec = _BinaryCodec;

  // src/websocket.js
  var _WebSocketManager = class _WebSocketManager {
    constructor(client, options = {}) {
      this.client = client;
      this.options = {
        reconnect: options.reconnect !== false,
        reconnectInterval: options.reconnectInterval || 1e3,
        reconnectMaxInterval: options.reconnectMaxInterval || 3e4,
        heartbeatInterval: options.heartbeatInterval || 3e4,
        ...options
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
      this.ws.binaryType = "arraybuffer";
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
        console.log("[Vango] WebSocket connected");
      }
      this.reconnectAttempts = 0;
      this._sendHandshake();
      this._startHeartbeat();
    }
    /**
     * Send handshake message (JSON, not binary)
     */
    _sendHandshake() {
      const handshake = {
        type: "handshake",
        version: "1.0",
        csrf: window.__VANGO_CSRF__ || "",
        session: this.sessionId || "",
        path: location.pathname,
        viewport: {
          width: window.innerWidth,
          height: window.innerHeight
        }
      };
      this.ws.send(JSON.stringify(handshake));
    }
    /**
     * Handle incoming message
     */
    _onMessage(event) {
      if (!this.connected) {
        try {
          const ack = JSON.parse(event.data);
          if (ack.type === "handshake_ack") {
            this.connected = true;
            this.sessionId = ack.session;
            this.client._onConnected();
            this._flushQueue();
            if (this.client.options.debug) {
              console.log("[Vango] Handshake complete, session:", this.sessionId);
            }
            return;
          }
        } catch (e) {
        }
      }
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
        console.log("[Vango] WebSocket closed:", event.code, event.reason);
      }
      if (wasConnected) {
        this.client._onDisconnected();
      }
      if (this.options.reconnect && !event.wasClean) {
        this._scheduleReconnect();
      }
    }
    /**
     * Handle WebSocket error
     */
    _onError(event) {
      if (this.client.options.debug) {
        console.error("[Vango] WebSocket error:", event);
      }
      this.client._onError(new Error("WebSocket error"));
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
        const buffer = new Uint8Array([2, 1]);
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
        this.ws.close(1e3, "Client closing");
        this.ws = null;
      }
    }
  };
  __name(_WebSocketManager, "WebSocketManager");
  var WebSocketManager = _WebSocketManager;

  // src/events.js
  var _EventCapture = class _EventCapture {
    constructor(client) {
      this.client = client;
      this.handlers = /* @__PURE__ */ new Map();
      this.debounceTimers = /* @__PURE__ */ new Map();
      this.scrollThrottled = /* @__PURE__ */ new Set();
    }
    /**
     * Attach event listeners to document
     */
    attach() {
      this._on("click", this._handleClick.bind(this));
      this._on("dblclick", this._handleDblClick.bind(this));
      this._on("input", this._handleInput.bind(this));
      this._on("change", this._handleChange.bind(this));
      this._on("submit", this._handleSubmit.bind(this));
      this._on("focus", this._handleFocus.bind(this), true);
      this._on("blur", this._handleBlur.bind(this), true);
      this._on("keydown", this._handleKeyDown.bind(this));
      this._on("keyup", this._handleKeyUp.bind(this));
      this._on("mouseenter", this._handleMouseEnter.bind(this), true);
      this._on("mouseleave", this._handleMouseLeave.bind(this), true);
      this._on("scroll", this._handleScroll.bind(this), true);
      this._on("click", this._handleLinkClick.bind(this));
      window.addEventListener("popstate", this._handlePopState.bind(this));
    }
    /**
     * Detach all event listeners
     */
    detach() {
      for (const [key, { handler, capture }] of this.handlers) {
        document.removeEventListener(key, handler, capture);
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
      this.handlers.set(`${eventType}-${capture}`, { handler, capture });
    }
    /**
     * Find closest element with data-hid
     */
    _findHidElement(target) {
      return target.closest("[data-hid]");
    }
    /**
     * Handle click event
     */
    _handleClick(event) {
      const el = this._findHidElement(event.target);
      if (!el)
        return;
      if (!el.hasAttribute("data-on-click"))
        return;
      event.preventDefault();
      this.client.optimistic.applyOptimistic(el, "click");
      this.client.sendEvent(EventType.CLICK, el.dataset.hid);
    }
    /**
     * Handle double-click event
     */
    _handleDblClick(event) {
      const el = this._findHidElement(event.target);
      if (!el || !el.hasAttribute("data-on-dblclick"))
        return;
      event.preventDefault();
      this.client.sendEvent(EventType.DBLCLICK, el.dataset.hid);
    }
    /**
     * Handle input event (debounced)
     */
    _handleInput(event) {
      const el = this._findHidElement(event.target);
      if (!el || !el.hasAttribute("data-on-input"))
        return;
      const hid = el.dataset.hid;
      const debounceMs = parseInt(el.dataset.debounce || "100", 10);
      if (this.debounceTimers.has(hid)) {
        clearTimeout(this.debounceTimers.get(hid));
      }
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
      if (!el || !el.hasAttribute("data-on-change"))
        return;
      let value;
      if (el.type === "checkbox") {
        value = el.checked ? "true" : "false";
      } else if (el.type === "radio") {
        value = el.checked ? el.value : "";
      } else if (el.tagName === "SELECT" && el.multiple) {
        value = Array.from(el.selectedOptions).map((o) => o.value).join(",");
      } else {
        value = el.value;
      }
      this.client.optimistic.applyOptimistic(el, "change");
      this.client.sendEvent(EventType.CHANGE, el.dataset.hid, { value });
    }
    /**
     * Handle form submit
     */
    _handleSubmit(event) {
      const form = event.target.closest("form[data-hid]");
      if (!form || !form.hasAttribute("data-on-submit"))
        return;
      event.preventDefault();
      const formData = new FormData(form);
      const fields = {};
      for (const [key, value] of formData.entries()) {
        fields[key] = String(value);
      }
      this.client.sendEvent(EventType.SUBMIT, form.dataset.hid, fields);
    }
    /**
     * Handle focus event
     */
    _handleFocus(event) {
      const el = this._findHidElement(event.target);
      if (!el || !el.hasAttribute("data-on-focus"))
        return;
      this.client.sendEvent(EventType.FOCUS, el.dataset.hid);
    }
    /**
     * Handle blur event
     */
    _handleBlur(event) {
      const el = this._findHidElement(event.target);
      if (!el || !el.hasAttribute("data-on-blur"))
        return;
      this.client.sendEvent(EventType.BLUR, el.dataset.hid);
    }
    /**
     * Handle keydown event
     */
    _handleKeyDown(event) {
      const el = this._findHidElement(event.target);
      if (!el || !el.hasAttribute("data-on-keydown"))
        return;
      const keyFilter = el.dataset.keyFilter;
      if (keyFilter && !this._matchesKeyFilter(event, keyFilter)) {
        return;
      }
      const shouldPrevent = el.dataset.preventDefault !== "false";
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
      if (!el || !el.hasAttribute("data-on-keyup"))
        return;
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
      if (!el || !el.hasAttribute("data-on-mouseenter"))
        return;
      this.client.sendEvent(EventType.MOUSEENTER, el.dataset.hid);
    }
    /**
     * Handle mouseleave event
     */
    _handleMouseLeave(event) {
      const el = this._findHidElement(event.target);
      if (!el || !el.hasAttribute("data-on-mouseleave"))
        return;
      this.client.sendEvent(EventType.MOUSELEAVE, el.dataset.hid);
    }
    /**
     * Handle scroll event (throttled)
     */
    _handleScroll(event) {
      const el = this._findHidElement(event.target);
      if (!el || !el.hasAttribute("data-on-scroll"))
        return;
      const hid = el.dataset.hid;
      const throttleMs = parseInt(el.dataset.throttle || "100", 10);
      if (this.scrollThrottled.has(hid)) {
        return;
      }
      this.scrollThrottled.add(hid);
      setTimeout(() => {
        this.scrollThrottled.delete(hid);
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
      const link = event.target.closest("a[href]");
      if (!link)
        return;
      const href = link.getAttribute("href");
      if (!href || href.startsWith("http") || href.startsWith("//")) {
        return;
      }
      if (link.hasAttribute("data-external") || link.target === "_blank") {
        return;
      }
      event.preventDefault();
      history.pushState(null, "", href);
      this.client.sendEvent(EventType.NAVIGATE, "nav", { path: href });
    }
    /**
     * Handle browser back/forward
     */
    _handlePopState(event) {
      this.client.sendEvent(EventType.NAVIGATE, "nav", { path: location.pathname });
    }
    /**
     * Check if event matches key filter
     * Format: "Enter" or "Ctrl+s" or "Meta+Enter"
     */
    _matchesKeyFilter(event, filter) {
      const parts = filter.split("+");
      const key = parts.pop().toLowerCase();
      const modifiers = new Set(parts.map((m) => m.toLowerCase()));
      if (event.key.toLowerCase() !== key) {
        return false;
      }
      const hasCtrl = modifiers.has("ctrl") || modifiers.has("control");
      const hasShift = modifiers.has("shift");
      const hasAlt = modifiers.has("alt");
      const hasMeta = modifiers.has("meta") || modifiers.has("cmd");
      if (hasCtrl !== event.ctrlKey)
        return false;
      if (hasShift !== event.shiftKey)
        return false;
      if (hasAlt !== event.altKey)
        return false;
      if (hasMeta !== event.metaKey)
        return false;
      return true;
    }
  };
  __name(_EventCapture, "EventCapture");
  var EventCapture = _EventCapture;

  // src/patches.js
  var _PatchApplier = class _PatchApplier {
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
      if (!el && patch.type !== PatchType.INSERT_NODE) {
        if (this.client.options.debug) {
          console.warn("[Vango] Node not found:", patch.hid);
        }
        return;
      }
      switch (patch.type) {
        case PatchType.SET_TEXT:
          this._setText(el, patch.value);
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
        case PatchType.TOGGLE_CLASS:
          el.classList.toggle(patch.className);
          break;
        case PatchType.SET_STYLE:
          el.style[patch.key] = patch.value;
          break;
        case PatchType.REMOVE_STYLE:
          el.style[patch.key] = "";
          break;
        case PatchType.INSERT_NODE:
          this._insertNode(patch.parentID, patch.index, patch.vnode);
          break;
        case PatchType.REMOVE_NODE:
          this._removeNode(el, patch.hid);
          break;
        case PatchType.MOVE_NODE:
          this._moveNode(el, patch.parentID, patch.index);
          break;
        case PatchType.REPLACE_NODE:
          this._replaceNode(el, patch.hid, patch.vnode);
          break;
        case PatchType.SET_VALUE:
          this._setValue(el, patch.value);
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
          el.scrollTo({
            left: patch.x,
            top: patch.y,
            behavior: patch.behavior === 1 ? "smooth" : "instant"
          });
          break;
        case PatchType.SET_DATA:
          el.dataset[patch.key] = patch.value;
          break;
        case PatchType.DISPATCH:
          this._dispatchEvent(el, patch.eventName, patch.detail);
          break;
        case PatchType.EVAL:
          this._evalCode(patch.code);
          break;
        default:
          if (this.client.options.debug) {
            console.warn("[Vango] Unknown patch type:", patch.type);
          }
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
      switch (key) {
        case "class":
          el.className = value;
          break;
        case "for":
          el.htmlFor = value;
          break;
        case "value":
          this._setValue(el, value);
          break;
        case "checked":
          el.checked = value === "true" || value === "";
          break;
        case "selected":
          el.selected = value === "true" || value === "";
          break;
        case "disabled":
        case "readonly":
        case "required":
        case "multiple":
        case "autofocus":
          if (value === "true" || value === "") {
            el.setAttribute(key, "");
          } else {
            el.removeAttribute(key);
          }
          break;
        default:
          el.setAttribute(key, value);
      }
    }
    /**
     * Remove attribute
     */
    _removeAttr(el, key) {
      switch (key) {
        case "class":
          el.className = "";
          break;
        case "for":
          el.htmlFor = "";
          break;
        case "value":
          el.value = "";
          break;
        case "checked":
          el.checked = false;
          break;
        case "selected":
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
      if (el.value === value)
        return;
      if (el.type === "text" || el.tagName === "TEXTAREA") {
        const start = el.selectionStart;
        const end = el.selectionEnd;
        el.value = value;
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
     * Insert node at index in parent
     */
    _insertNode(parentHid, index, vnode) {
      const parentEl = this.client.getNode(parentHid);
      if (!parentEl) {
        if (this.client.options.debug) {
          console.warn("[Vango] Parent node not found:", parentHid);
        }
        return;
      }
      const newEl = this._createNode(vnode);
      if (index >= parentEl.children.length) {
        parentEl.appendChild(newEl);
      } else {
        parentEl.insertBefore(newEl, parentEl.children[index]);
      }
    }
    /**
     * Remove node
     */
    _removeNode(el, hid) {
      this.client.hooks.destroyForNode(el);
      this.client.unregisterNode(hid);
      el.querySelectorAll("[data-hid]").forEach((child) => {
        this.client.hooks.destroyForNode(child);
        this.client.unregisterNode(child.dataset.hid);
      });
      el.remove();
    }
    /**
     * Move node to new position
     */
    _moveNode(el, parentHid, index) {
      const parentEl = this.client.getNode(parentHid);
      if (!parentEl) {
        if (this.client.options.debug) {
          console.warn("[Vango] Parent node not found:", parentHid);
        }
        return;
      }
      if (index >= parentEl.children.length) {
        parentEl.appendChild(el);
      } else {
        parentEl.insertBefore(el, parentEl.children[index]);
      }
    }
    /**
     * Replace node
     */
    _replaceNode(el, hid, vnode) {
      this.client.hooks.destroyForNode(el);
      this.client.unregisterNode(hid);
      el.querySelectorAll("[data-hid]").forEach((child) => {
        this.client.hooks.destroyForNode(child);
        this.client.unregisterNode(child.dataset.hid);
      });
      const newEl = this._createNode(vnode);
      el.replaceWith(newEl);
    }
    /**
     * Create DOM node from VNode
     */
    _createNode(vnode) {
      switch (vnode.type) {
        case "element":
          return this._createElement(vnode);
        case "text":
          return document.createTextNode(vnode.text);
        case "fragment":
          return this._createFragment(vnode);
        default:
          if (this.client.options.debug) {
            console.warn("[Vango] Unknown vnode type:", vnode.type);
          }
          return document.createTextNode("");
      }
    }
    /**
     * Create element from VNode
     */
    _createElement(vnode) {
      const el = document.createElement(vnode.tag);
      for (const [key, value] of Object.entries(vnode.attrs || {})) {
        this._setAttr(el, key, value);
      }
      if (vnode.hid) {
        el.dataset.hid = vnode.hid;
        this.client.registerNode(vnode.hid, el);
      }
      for (const child of vnode.children || []) {
        el.appendChild(this._createNode(child));
      }
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
    /**
     * Dispatch custom event
     */
    _dispatchEvent(el, eventName, detail) {
      let parsedDetail = null;
      if (detail) {
        try {
          parsedDetail = JSON.parse(detail);
        } catch (e) {
          parsedDetail = detail;
        }
      }
      const event = new CustomEvent(eventName, {
        detail: parsedDetail,
        bubbles: true,
        cancelable: true
      });
      el.dispatchEvent(event);
    }
    /**
     * Evaluate JavaScript code (use sparingly!)
     */
    _evalCode(code) {
      try {
        new Function(code)();
      } catch (e) {
        console.error("[Vango] Eval error:", e);
      }
    }
  };
  __name(_PatchApplier, "PatchApplier");
  var PatchApplier = _PatchApplier;

  // src/optimistic.js
  var _OptimisticUpdates = class _OptimisticUpdates {
    constructor(client) {
      this.client = client;
      this.pending = /* @__PURE__ */ new Map();
    }
    /**
     * Apply optimistic update based on element data attributes
     */
    applyOptimistic(el, eventType) {
      const hid = el.dataset.hid;
      if (!hid)
        return;
      const optimisticClass = el.dataset.optimisticClass;
      if (optimisticClass) {
        this._applyClassOptimistic(el, hid, optimisticClass);
      }
      const optimisticText = el.dataset.optimisticText;
      if (optimisticText) {
        this._applyTextOptimistic(el, hid, optimisticText);
      }
      const optimisticAttr = el.dataset.optimisticAttr;
      const optimisticValue = el.dataset.optimisticValue;
      if (optimisticAttr && optimisticValue !== void 0) {
        this._applyAttrOptimistic(el, hid, optimisticAttr, optimisticValue);
      }
      const parentOptimisticClass = el.dataset.optimisticParentClass;
      if (parentOptimisticClass && el.parentElement) {
        const parentHid = el.parentElement.dataset.hid || `parent-${hid}`;
        this._applyClassOptimistic(el.parentElement, parentHid, parentOptimisticClass);
      }
    }
    /**
     * Apply optimistic class toggle
     * Format: "classname" or "classname:add" or "classname:remove" or "classname:toggle"
     */
    _applyClassOptimistic(el, hid, classConfig) {
      const [className, action = "toggle"] = classConfig.split(":");
      const original = el.classList.contains(className);
      switch (action) {
        case "add":
          el.classList.add(className);
          break;
        case "remove":
          el.classList.remove(className);
          break;
        case "toggle":
        default:
          el.classList.toggle(className);
      }
      this._trackPending(hid, "class", className, original);
    }
    /**
     * Apply optimistic text change
     */
    _applyTextOptimistic(el, hid, text) {
      const original = el.textContent;
      el.textContent = text;
      this._trackPending(hid, "text", null, original);
    }
    /**
     * Apply optimistic attribute change
     */
    _applyAttrOptimistic(el, hid, attr, value) {
      const original = el.getAttribute(attr);
      if (value === "null" || value === "") {
        el.removeAttribute(attr);
      } else {
        el.setAttribute(attr, value);
      }
      this._trackPending(hid, "attr", attr, original);
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
        if (!el)
          continue;
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
        case "class":
          if (update.original) {
            el.classList.add(update.key);
          } else {
            el.classList.remove(update.key);
          }
          break;
        case "text":
          el.textContent = update.original;
          break;
        case "attr":
          if (update.original === null) {
            el.removeAttribute(update.key);
          } else {
            el.setAttribute(update.key, update.original);
          }
          break;
      }
    }
  };
  __name(_OptimisticUpdates, "OptimisticUpdates");
  var OptimisticUpdates = _OptimisticUpdates;

  // src/hooks/sortable.js
  var _SortableHook = class _SortableHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.config = config;
      this.pushEvent = pushEvent;
      this.animation = config.animation || 150;
      this.handle = config.handle || null;
      this.ghostClass = config.ghostClass || "sortable-ghost";
      this.dragClass = config.dragClass || "sortable-drag";
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
      this.el.addEventListener("mousedown", this._onMouseDown);
      document.addEventListener("mousemove", this._onMouseMove);
      document.addEventListener("mouseup", this._onMouseUp);
      this._onTouchStart = this._handleTouchStart.bind(this);
      this._onTouchMove = this._handleTouchMove.bind(this);
      this._onTouchEnd = this._handleTouchEnd.bind(this);
      this.el.addEventListener("touchstart", this._onTouchStart, { passive: false });
      document.addEventListener("touchmove", this._onTouchMove, { passive: false });
      document.addEventListener("touchend", this._onTouchEnd);
    }
    _unbindEvents() {
      this.el.removeEventListener("mousedown", this._onMouseDown);
      document.removeEventListener("mousemove", this._onMouseMove);
      document.removeEventListener("mouseup", this._onMouseUp);
      this.el.removeEventListener("touchstart", this._onTouchStart);
      document.removeEventListener("touchmove", this._onTouchMove);
      document.removeEventListener("touchend", this._onTouchEnd);
    }
    _handleMouseDown(e) {
      const item = this._findItem(e.target);
      if (!item)
        return;
      if (this.handle && !e.target.closest(this.handle))
        return;
      e.preventDefault();
      this._startDrag(item, e.clientY);
    }
    _handleTouchStart(e) {
      const item = this._findItem(e.target);
      if (!item)
        return;
      if (this.handle && !e.target.closest(this.handle))
        return;
      e.preventDefault();
      const touch = e.touches[0];
      this._startDrag(item, touch.clientY);
    }
    _findItem(target) {
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
      this.ghost = item.cloneNode(true);
      this.ghost.classList.add(this.ghostClass);
      this.ghost.style.position = "fixed";
      this.ghost.style.zIndex = "9999";
      this.ghost.style.width = `${item.offsetWidth}px`;
      this.ghost.style.left = `${item.getBoundingClientRect().left}px`;
      this.ghost.style.top = `${item.getBoundingClientRect().top}px`;
      this.ghost.style.pointerEvents = "none";
      this.ghost.style.opacity = "0.8";
      document.body.appendChild(this.ghost);
      item.classList.add(this.dragClass);
      item.style.opacity = "0.4";
    }
    _handleMouseMove(e) {
      if (!this.dragging)
        return;
      e.preventDefault();
      this._updateDrag(e.clientY);
    }
    _handleTouchMove(e) {
      if (!this.dragging)
        return;
      e.preventDefault();
      const touch = e.touches[0];
      this._updateDrag(touch.clientY);
    }
    _updateDrag(y) {
      var _a;
      const deltaY = y - this.startY;
      const startTop = (_a = this.el.children[this.startIndex]) == null ? void 0 : _a.getBoundingClientRect().top;
      if (startTop !== void 0 && this.ghost) {
        this.ghost.style.top = `${startTop + deltaY}px`;
      }
      const children = Array.from(this.el.children);
      const currentIndex = children.indexOf(this.dragging);
      for (let i = 0; i < children.length; i++) {
        if (i === currentIndex)
          continue;
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
      if (!this.dragging)
        return;
      this._endDrag();
    }
    _handleTouchEnd(e) {
      if (!this.dragging)
        return;
      this._endDrag();
    }
    _endDrag() {
      const endIndex = Array.from(this.el.children).indexOf(this.dragging);
      this.dragging.classList.remove(this.dragClass);
      this.dragging.style.opacity = "";
      if (this.ghost) {
        this.ghost.remove();
      }
      if (endIndex !== this.startIndex) {
        const id = this.dragging.dataset.id || this.dragging.dataset.hid;
        this.pushEvent("reorder", {
          id,
          fromIndex: this.startIndex,
          toIndex: endIndex
        });
      }
      this.dragging = null;
      this.ghost = null;
    }
  };
  __name(_SortableHook, "SortableHook");
  var SortableHook = _SortableHook;

  // src/hooks/draggable.js
  var _DraggableHook = class _DraggableHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.config = config;
      this.pushEvent = pushEvent;
      this.axis = config.axis || "both";
      this.handle = config.handle || null;
      this.bounds = config.bounds || null;
      this.dragging = false;
      this.startX = 0;
      this.startY = 0;
      this.currentX = 0;
      this.currentY = 0;
      this._bindEvents();
    }
    updated(el, config, pushEvent) {
      this.config = config;
      this.pushEvent = pushEvent;
      this.axis = config.axis || "both";
      this.bounds = config.bounds || null;
    }
    destroyed(el) {
      this._unbindEvents();
    }
    _bindEvents() {
      this._onMouseDown = this._handleMouseDown.bind(this);
      this._onMouseMove = this._handleMouseMove.bind(this);
      this._onMouseUp = this._handleMouseUp.bind(this);
      const handleEl = this.handle ? this.el.querySelector(this.handle) : this.el;
      if (handleEl) {
        handleEl.addEventListener("mousedown", this._onMouseDown);
      }
      document.addEventListener("mousemove", this._onMouseMove);
      document.addEventListener("mouseup", this._onMouseUp);
      this._onTouchStart = this._handleTouchStart.bind(this);
      this._onTouchMove = this._handleTouchMove.bind(this);
      this._onTouchEnd = this._handleTouchEnd.bind(this);
      if (handleEl) {
        handleEl.addEventListener("touchstart", this._onTouchStart, { passive: false });
      }
      document.addEventListener("touchmove", this._onTouchMove, { passive: false });
      document.addEventListener("touchend", this._onTouchEnd);
    }
    _unbindEvents() {
      const handleEl = this.handle ? this.el.querySelector(this.handle) : this.el;
      if (handleEl) {
        handleEl.removeEventListener("mousedown", this._onMouseDown);
        handleEl.removeEventListener("touchstart", this._onTouchStart);
      }
      document.removeEventListener("mousemove", this._onMouseMove);
      document.removeEventListener("mouseup", this._onMouseUp);
      document.removeEventListener("touchmove", this._onTouchMove);
      document.removeEventListener("touchend", this._onTouchEnd);
    }
    _handleMouseDown(e) {
      e.preventDefault();
      this._startDrag(e.clientX, e.clientY);
    }
    _handleTouchStart(e) {
      e.preventDefault();
      const touch = e.touches[0];
      this._startDrag(touch.clientX, touch.clientY);
    }
    _startDrag(x, y) {
      this.dragging = true;
      this.startX = x - this.currentX;
      this.startY = y - this.currentY;
      this.el.classList.add("dragging");
    }
    _handleMouseMove(e) {
      if (!this.dragging)
        return;
      e.preventDefault();
      this._updatePosition(e.clientX, e.clientY);
    }
    _handleTouchMove(e) {
      if (!this.dragging)
        return;
      e.preventDefault();
      const touch = e.touches[0];
      this._updatePosition(touch.clientX, touch.clientY);
    }
    _updatePosition(x, y) {
      let newX = x - this.startX;
      let newY = y - this.startY;
      if (this.axis === "x") {
        newY = this.currentY;
      } else if (this.axis === "y") {
        newX = this.currentX;
      }
      if (this.bounds) {
        const bounds = this._getBounds();
        newX = Math.max(bounds.minX, Math.min(bounds.maxX, newX));
        newY = Math.max(bounds.minY, Math.min(bounds.maxY, newY));
      }
      this.currentX = newX;
      this.currentY = newY;
      this.el.style.transform = `translate(${newX}px, ${newY}px)`;
    }
    _getBounds() {
      const rect = this.el.getBoundingClientRect();
      let container;
      if (this.bounds === "parent") {
        container = this.el.parentElement.getBoundingClientRect();
      } else {
        container = {
          left: 0,
          top: 0,
          right: window.innerWidth,
          bottom: window.innerHeight
        };
      }
      return {
        minX: container.left - rect.left + this.currentX,
        maxX: container.right - rect.right + this.currentX,
        minY: container.top - rect.top + this.currentY,
        maxY: container.bottom - rect.bottom + this.currentY
      };
    }
    _handleMouseUp(e) {
      if (!this.dragging)
        return;
      this._endDrag();
    }
    _handleTouchEnd(e) {
      if (!this.dragging)
        return;
      this._endDrag();
    }
    _endDrag() {
      this.dragging = false;
      this.el.classList.remove("dragging");
      this.pushEvent("position", {
        x: this.currentX,
        y: this.currentY
      });
    }
  };
  __name(_DraggableHook, "DraggableHook");
  var DraggableHook = _DraggableHook;

  // src/hooks/tooltip.js
  var _TooltipHook = class _TooltipHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.config = config;
      this.tooltip = null;
      this.content = config.content || "";
      this.placement = config.placement || "top";
      this.delay = config.delay || 200;
      this._bindEvents();
    }
    updated(el, config, pushEvent) {
      this.content = config.content || "";
      this.placement = config.placement || "top";
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
      this.el.addEventListener("mouseenter", this._onMouseEnter);
      this.el.addEventListener("mouseleave", this._onMouseLeave);
    }
    _unbindEvents() {
      this.el.removeEventListener("mouseenter", this._onMouseEnter);
      this.el.removeEventListener("mouseleave", this._onMouseLeave);
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
      if (!this.content)
        return;
      this.tooltip = document.createElement("div");
      this.tooltip.className = "vango-tooltip";
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
      this._position();
    }
    _position() {
      if (!this.tooltip)
        return;
      const rect = this.el.getBoundingClientRect();
      const tipRect = this.tooltip.getBoundingClientRect();
      let top, left;
      switch (this.placement) {
        case "top":
          top = rect.top - tipRect.height - 8;
          left = rect.left + (rect.width - tipRect.width) / 2;
          break;
        case "bottom":
          top = rect.bottom + 8;
          left = rect.left + (rect.width - tipRect.width) / 2;
          break;
        case "left":
          top = rect.top + (rect.height - tipRect.height) / 2;
          left = rect.left - tipRect.width - 8;
          break;
        case "right":
          top = rect.top + (rect.height - tipRect.height) / 2;
          left = rect.right + 8;
          break;
        default:
          top = rect.top - tipRect.height - 8;
          left = rect.left + (rect.width - tipRect.width) / 2;
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
  };
  __name(_TooltipHook, "TooltipHook");
  var TooltipHook = _TooltipHook;

  // src/hooks/dropdown.js
  var _DropdownHook = class _DropdownHook {
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
      setTimeout(() => {
        document.addEventListener("click", this._onClickOutside);
        document.addEventListener("keydown", this._onKeyDown);
      }, 0);
    }
    _unbindEvents() {
      document.removeEventListener("click", this._onClickOutside);
      document.removeEventListener("keydown", this._onKeyDown);
    }
    _handleClickOutside(e) {
      if (!this.closeOnClickOutside)
        return;
      if (!this.el.contains(e.target)) {
        this.pushEvent("close", {});
      }
    }
    _handleKeyDown(e) {
      if (!this.closeOnEscape)
        return;
      if (e.key === "Escape") {
        e.preventDefault();
        this.pushEvent("close", {});
      }
    }
  };
  __name(_DropdownHook, "DropdownHook");
  var DropdownHook = _DropdownHook;

  // src/hooks/manager.js
  var _HookManager = class _HookManager {
    constructor(client) {
      this.client = client;
      this.instances = /* @__PURE__ */ new Map();
      this.hooks = {
        "Sortable": SortableHook,
        "Draggable": DraggableHook,
        "Tooltip": TooltipHook,
        "Dropdown": DropdownHook
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
      document.querySelectorAll("[data-hook]").forEach((el) => {
        this.initializeForNode(el);
      });
    }
    /**
     * Update hooks after DOM changes
     */
    updateFromDOM() {
      document.querySelectorAll("[data-hook]").forEach((el) => {
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
      if (!hookName)
        return;
      const HookClass = this.hooks[hookName];
      if (!HookClass) {
        if (this.client.options.debug) {
          console.warn("[Vango] Unknown hook:", hookName);
        }
        return;
      }
      const hid = el.dataset.hid;
      if (!hid) {
        if (this.client.options.debug) {
          console.warn("[Vango] Hook element must have data-hid");
        }
        return;
      }
      if (this.instances.has(hid)) {
        return;
      }
      let config = {};
      try {
        if (el.dataset.hookConfig) {
          config = JSON.parse(el.dataset.hookConfig);
        }
      } catch (e) {
        if (this.client.options.debug) {
          console.warn("[Vango] Invalid hook config:", e);
        }
      }
      const pushEvent = /* @__PURE__ */ __name((eventName, data = {}) => {
        this.client.sendHookEvent(hid, eventName, data);
      }, "pushEvent");
      const instance = new HookClass();
      instance.mounted(el, config, pushEvent);
      this.instances.set(hid, { hook: HookClass, instance, el });
    }
    /**
     * Destroy hook for a specific node
     */
    destroyForNode(el) {
      var _a;
      const hid = (_a = el.dataset) == null ? void 0 : _a.hid;
      if (!hid)
        return;
      const entry = this.instances.get(hid);
      if (entry) {
        if (entry.instance.destroyed) {
          entry.instance.destroyed(entry.el);
        }
        this.instances.delete(hid);
      }
    }
    /**
     * Destroy all hooks
     */
    destroyAll() {
      for (const [hid, entry] of this.instances) {
        if (entry.instance.destroyed) {
          entry.instance.destroyed(entry.el);
        }
      }
      this.instances.clear();
    }
    /**
     * Update hook config
     */
    updateConfig(hid, config) {
      const entry = this.instances.get(hid);
      if (entry && entry.instance.updated) {
        const pushEvent = /* @__PURE__ */ __name((eventName, data = {}) => {
          this.client.sendHookEvent(hid, eventName, data);
        }, "pushEvent");
        entry.instance.updated(entry.el, config, pushEvent);
      }
    }
  };
  __name(_HookManager, "HookManager");
  var HookManager = _HookManager;

  // src/index.js
  var FrameType = {
    EVENT: 0,
    PATCHES: 1,
    CONTROL: 2,
    ERROR: 3
  };
  var ControlType = {
    PING: 1,
    PONG: 2,
    RESYNC: 3,
    CLOSE: 4
  };
  var _VangoClient = class _VangoClient {
    constructor(options = {}) {
      this.options = {
        wsUrl: options.wsUrl || this._defaultWsUrl(),
        reconnect: options.reconnect !== false,
        reconnectInterval: options.reconnectInterval || 1e3,
        reconnectMaxInterval: options.reconnectMaxInterval || 3e4,
        heartbeatInterval: options.heartbeatInterval || 3e4,
        debug: options.debug || false,
        ...options
      };
      this.codec = new BinaryCodec();
      this.nodeMap = /* @__PURE__ */ new Map();
      this.connected = false;
      this.seq = 0;
      this.wsManager = new WebSocketManager(this, this.options);
      this.patchApplier = new PatchApplier(this);
      this.eventCapture = new EventCapture(this);
      this.optimistic = new OptimisticUpdates(this);
      this.hooks = new HookManager(this);
      this.onConnect = options.onConnect || (() => {
      });
      this.onDisconnect = options.onDisconnect || (() => {
      });
      this.onError = options.onError || ((err) => console.error("[Vango]", err));
      this._buildNodeMap();
      this.wsManager.connect(this.options.wsUrl);
      this.eventCapture.attach();
      this.hooks.initializeFromDOM();
    }
    /**
     * Build initial map of data-hid -> DOM node
     */
    _buildNodeMap() {
      document.querySelectorAll("[data-hid]").forEach((el) => {
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
      const protocol = location.protocol === "https:" ? "wss:" : "ws:";
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
      if (buffer.length === 0)
        return;
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
            console.warn("[Vango] Unknown frame type:", frameType);
          }
      }
    }
    /**
     * Handle patches frame
     */
    _handlePatches(buffer) {
      const { seq, patches } = this.codec.decodePatches(buffer);
      if (this.options.debug) {
        console.log("[Vango] Applying", patches.length, "patches (seq:", seq, ")");
      }
      this.optimistic.clearPending();
      this.patchApplier.apply(patches);
      this.hooks.updateFromDOM();
    }
    /**
     * Handle control message
     */
    _handleControl(buffer) {
      if (buffer.length === 0)
        return;
      const controlType = buffer[0];
      switch (controlType) {
        case ControlType.PONG:
          if (this.options.debug) {
            console.log("[Vango] Pong received");
          }
          break;
        case ControlType.RESYNC:
          this._handleResync();
          break;
        case ControlType.CLOSE:
          this.wsManager.close();
          break;
      }
    }
    /**
     * Handle resync (full page refresh needed)
     */
    _handleResync() {
      if (this.options.debug) {
        console.log("[Vango] Resync requested, reloading page");
      }
      location.reload();
    }
    /**
     * Handle server error
     */
    _handleServerError(buffer) {
      if (buffer.length < 3)
        return;
      const code = buffer[0] << 8 | buffer[1];
      const fatal = buffer[2] === 1;
      const messageBytes = buffer.slice(3);
      const { value: message } = this.codec.decodeString(messageBytes, 0);
      const errorMessages = {
        1: "Session expired",
        2: "Invalid event",
        3: "Rate limited",
        4: "Server error",
        5: "Handler panic",
        6: "Invalid CSRF"
      };
      const errorMessage = errorMessages[code] || message || `Unknown error: ${code}`;
      const error = new Error(errorMessage);
      error.code = code;
      error.fatal = fatal;
      this.onError(error);
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
          console.log("[Vango] Not connected, queueing event");
        }
      }
      this.seq++;
      const eventBuffer = this.codec.encodeEvent(this.seq, type, hid, data);
      const frame = new Uint8Array(1 + eventBuffer.length);
      frame[0] = FrameType.EVENT;
      frame.set(eventBuffer, 1);
      this.wsManager.send(frame);
      if (this.options.debug) {
        console.log("[Vango] Sent event:", { type, hid, data, seq: this.seq });
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
  };
  __name(_VangoClient, "VangoClient");
  var VangoClient = _VangoClient;
  function init() {
    if (window.__vango__) {
      return;
    }
    const script = document.currentScript || document.querySelector("script[data-vango]");
    const options = {};
    if (script) {
      if (script.dataset.wsUrl)
        options.wsUrl = script.dataset.wsUrl;
      if (script.dataset.debug)
        options.debug = script.dataset.debug === "true";
    }
    window.__vango__ = new VangoClient(options);
  }
  __name(init, "init");
  if (typeof document !== "undefined") {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", init);
    } else {
      init();
    }
  }
  var src_default = VangoClient;
  return __toCommonJS(src_exports);
})();
//# sourceMappingURL=vango.js.map
