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
    ConnectionManager: () => ConnectionManager,
    ConnectionState: () => ConnectionState,
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
    // NOTE: EVAL (0x21) has been REMOVED for security. Server never sends it.
    // URL operations (Phase 12: URLParam 2.0)
    URL_PUSH: 48,
    URL_REPLACE: 49,
    // Navigation operations (full route navigation)
    NAV_PUSH: 50,
    NAV_REPLACE: 51
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
        case PatchType.URL_PUSH:
        case PatchType.URL_REPLACE: {
          const { value: count, bytesRead: countBytes } = this.decodeUvarint(buffer, offset);
          offset += countBytes;
          patch.params = {};
          for (let i = 0; i < count; i++) {
            const { value: key, bytesRead: keyBytes } = this.decodeString(buffer, offset);
            offset += keyBytes;
            const { value, bytesRead: valueBytes } = this.decodeString(buffer, offset);
            offset += valueBytes;
            patch.params[key] = value;
          }
          patch.op = patch.type;
          break;
        }
        case PatchType.NAV_PUSH:
        case PatchType.NAV_REPLACE: {
          const { value: path, bytesRead: pathBytes } = this.decodeString(buffer, offset);
          offset += pathBytes;
          patch.path = path;
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
     * Per spec section 3.9.3, lines 1091-1103
     */
    encodeKeyboardEvent(parts, data) {
      parts.push(this.encodeString((data == null ? void 0 : data.key) || ""));
      parts.push(this.encodeString((data == null ? void 0 : data.code) || ""));
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
      parts.push(new Uint8Array([(data == null ? void 0 : data.repeat) ? 1 : 0]));
      parts.push(new Uint8Array([(data == null ? void 0 : data.location) || 0]));
    }
    /**
     * Encode mouse event
     * Per spec section 3.9.3, lines 1059-1075
     */
    encodeMouseEvent(parts, data) {
      parts.push(this.encodeSvarint((data == null ? void 0 : data.clientX) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.clientY) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.pageX) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.pageY) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.offsetX) || 0));
      parts.push(this.encodeSvarint((data == null ? void 0 : data.offsetY) || 0));
      parts.push(new Uint8Array([(data == null ? void 0 : data.button) || 0]));
      parts.push(new Uint8Array([(data == null ? void 0 : data.buttons) || 0]));
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
     * Per spec section 3.9.3, lines 1212-1225
     */
    encodeTouchEvent(parts, data) {
      const touches = (data == null ? void 0 : data.touches) || [];
      parts.push(this.encodeUvarint(touches.length));
      for (const touch of touches) {
        parts.push(this.encodeSvarint(touch.identifier || touch.id || 0));
        parts.push(this.encodeSvarint(touch.clientX || 0));
        parts.push(this.encodeSvarint(touch.clientY || 0));
        parts.push(this.encodeSvarint(touch.pageX || 0));
        parts.push(this.encodeSvarint(touch.pageY || 0));
      }
      const targetTouches = (data == null ? void 0 : data.targetTouches) || [];
      parts.push(this.encodeUvarint(targetTouches.length));
      for (const touch of targetTouches) {
        parts.push(this.encodeSvarint(touch.identifier || touch.id || 0));
        parts.push(this.encodeSvarint(touch.clientX || 0));
        parts.push(this.encodeSvarint(touch.clientY || 0));
        parts.push(this.encodeSvarint(touch.pageX || 0));
        parts.push(this.encodeSvarint(touch.pageY || 0));
      }
      const changedTouches = (data == null ? void 0 : data.changedTouches) || [];
      parts.push(this.encodeUvarint(changedTouches.length));
      for (const touch of changedTouches) {
        parts.push(this.encodeSvarint(touch.identifier || touch.id || 0));
        parts.push(this.encodeSvarint(touch.clientX || 0));
        parts.push(this.encodeSvarint(touch.clientY || 0));
        parts.push(this.encodeSvarint(touch.pageX || 0));
        parts.push(this.encodeSvarint(touch.pageY || 0));
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
    /**
     * Encode ClientHello for handshake (raw payload, no frame header)
     * Format: [major:1][minor:1][csrf:string][sessionID:string][lastSeq:4][viewportW:2][viewportH:2][tzOffset:2]
     */
    encodeClientHello(options = {}) {
      const parts = [];
      parts.push(new Uint8Array([2, 0]));
      parts.push(this.encodeString(options.csrf || ""));
      parts.push(this.encodeString(options.sessionId || ""));
      parts.push(this.encodeUint32(options.lastSeq || 0));
      parts.push(this.encodeUint16(options.viewportW || window.innerWidth));
      parts.push(this.encodeUint16(options.viewportH || window.innerHeight));
      const tzOffset = (/* @__PURE__ */ new Date()).getTimezoneOffset();
      parts.push(this.encodeInt16(-tzOffset));
      return concat(parts);
    }
    /**
     * Encode ClientHello wrapped in frame header for consistent framing.
     * Format: [type:1][flags:1][len:2][payload...]
     *
     * Per spec: All protocol messages should use the same frame format.
     */
    encodeClientHelloFrame(options = {}) {
      const payload = this.encodeClientHello(options);
      const frame = new Uint8Array(4 + payload.length);
      frame[0] = 0;
      frame[1] = 0;
      frame[2] = payload.length >> 8 & 255;
      frame[3] = payload.length & 255;
      frame.set(payload, 4);
      return frame;
    }
    /**
     * Decode ServerHello from handshake response
     * Frame format: [type:1][flags:1][len:2][payload...]
     * ServerHello payload: [status:1][sessionID:string][nextSeq:4][serverTime:8][flags:2]
     */
    decodeServerHello(buffer) {
      if (buffer.length < 5) {
        return { error: "Buffer too short" };
      }
      const frameType = buffer[0];
      if (frameType !== 0) {
        return { error: `Unexpected frame type: ${frameType}` };
      }
      let offset = 4;
      const status = buffer[offset++];
      const { value: sessionId, bytesRead: sessionBytes } = this.decodeString(buffer, offset);
      offset += sessionBytes;
      const nextSeq = this.decodeUint32(buffer, offset);
      offset += 4;
      const serverTime = this.decodeUint64(buffer, offset);
      offset += 8;
      const flags = this.decodeUint16(buffer, offset);
      return {
        status,
        sessionId,
        nextSeq,
        serverTime,
        flags,
        ok: status === 0
      };
    }
    /**
     * Encode uint16 big-endian (matches Go protocol)
     */
    encodeUint16(value) {
      return new Uint8Array([value >> 8 & 255, value & 255]);
    }
    /**
     * Decode uint16 big-endian (matches Go protocol)
     */
    decodeUint16(buffer, offset) {
      return buffer[offset] << 8 | buffer[offset + 1];
    }
    /**
     * Encode int16 big-endian (matches Go protocol)
     */
    encodeInt16(value) {
      return this.encodeUint16(value & 65535);
    }
    /**
     * Encode uint32 big-endian (matches Go protocol)
     */
    encodeUint32(value) {
      return new Uint8Array([
        value >> 24 & 255,
        value >> 16 & 255,
        value >> 8 & 255,
        value & 255
      ]);
    }
    /**
     * Decode uint32 big-endian (matches Go protocol)
     */
    decodeUint32(buffer, offset) {
      return buffer[offset] << 24 | buffer[offset + 1] << 16 | buffer[offset + 2] << 8 | buffer[offset + 3];
    }
    /**
     * Decode uint64 big-endian (returns as Number, may lose precision for large values)
     * Matches Go protocol encoding.
     */
    decodeUint64(buffer, offset) {
      const high = this.decodeUint32(buffer, offset);
      const low = this.decodeUint32(buffer, offset + 4);
      return high * 4294967296 + low;
    }
    /**
     * Encode ACK payload
     * Format: [lastSeq:varint][window:varint]
     * Matches vango/pkg/protocol/ack.go
     */
    encodeAck(lastSeq, window2 = 100) {
      return concat([
        this.encodeUvarint(lastSeq),
        this.encodeUvarint(window2)
      ]);
    }
    /**
     * Encode ResyncRequest control payload
     * Format: [controlType:1][lastSeq:varint]
     * Matches vango/pkg/protocol/control.go ControlResyncRequest (0x10)
     */
    encodeResyncRequest(lastSeq) {
      return concat([
        new Uint8Array([16]),
        // ControlResyncRequest
        this.encodeUvarint(lastSeq)
      ]);
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
     * Send binary ClientHello handshake wrapped in frame header.
     * Per spec: All protocol messages use consistent framing.
     */
    _sendHandshake() {
      const helloFrame = this.client.codec.encodeClientHelloFrame({
        csrf: this._getCSRFToken(),
        sessionId: this.sessionId || "",
        viewportW: window.innerWidth,
        viewportH: window.innerHeight
      });
      this.ws.send(helloFrame);
      if (this.client.options.debug) {
        console.log("[Vango] Sent framed ClientHello");
      }
    }
    /**
     * Get CSRF token from window global or cookie (Double Submit Cookie pattern)
     */
    _getCSRFToken() {
      if (window.__VANGO_CSRF__) {
        return window.__VANGO_CSRF__;
      }
      const match = document.cookie.match(/(?:^|;\s*)__vango_csrf=([^;]*)/);
      return match ? decodeURIComponent(match[1]) : "";
    }
    /**
     * Handle incoming message
     */
    _onMessage(event) {
      if (!(event.data instanceof ArrayBuffer)) {
        if (this.client.options.debug) {
          console.warn("[Vango] Received non-binary message:", event.data);
        }
        return;
      }
      const buffer = new Uint8Array(event.data);
      if (!this.handshakeComplete) {
        const hello = this.client.codec.decodeServerHello(buffer);
        if (hello.error) {
          if (this.client.options.debug) {
            console.error("[Vango] Handshake error:", hello.error);
          }
          this.client._onError(new Error(hello.error));
          return;
        }
        if (!hello.ok) {
          const errorMessages = {
            1: "Version mismatch",
            2: "Invalid CSRF token",
            3: "Session expired",
            4: "Server busy",
            5: "Upgrade required",
            6: "Invalid format",
            7: "Not authorized",
            8: "Internal error"
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
        this._flushQueue();
        if (this.client.options.debug) {
          console.log("[Vango] Handshake complete, session:", this.sessionId);
        }
        return;
      }
      this.client._handleBinaryMessage(buffer);
    }
    /**
     * Handle WebSocket close
     *
     * Per spec Section 9.6.3 (Connection Loss During Navigation):
     * If WebSocket closes while awaiting navigation response,
     * complete the navigation via location.assign(pendingPath).
     */
    _onClose(event) {
      var _a;
      const wasConnected = this.connected;
      this.connected = false;
      this.handshakeComplete = false;
      this._stopHeartbeat();
      if (this.client.options.debug) {
        console.log("[Vango] WebSocket closed:", event.code, event.reason);
      }
      const pendingPath = (_a = this.client.eventCapture) == null ? void 0 : _a.pendingNavPath;
      if (wasConnected && pendingPath) {
        console.log("[Vango] Connection lost during navigation, completing via location.assign:", pendingPath);
        location.assign(pendingPath);
        return;
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
        const timestamp = Date.now();
        const payload = new Uint8Array(9);
        payload[0] = 1;
        let ts = timestamp;
        for (let i = 0; i < 8; i++) {
          payload[1 + i] = ts & 255;
          ts = Math.floor(ts / 256);
        }
        const frame = new Uint8Array(4 + payload.length);
        frame[0] = 3;
        frame[1] = 0;
        frame[2] = payload.length >> 8 & 255;
        frame[3] = payload.length & 255;
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
      this.prefetchedPaths = /* @__PURE__ */ new Set();
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
      this._on("mouseenter", this._handlePrefetch.bind(this), true);
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
      if (!target || !target.closest) {
        return null;
      }
      return target.closest("[data-hid]");
    }
    _closest(target, selector) {
      if (!target || typeof target.closest !== "function") {
        return null;
      }
      return target.closest(selector);
    }
    /**
     * Check if element has an event in its data-ve attribute.
     * Parses data-ve="click,input,change" format per spec Section 5.2.
     */
    _hasEvent(el, eventName) {
      const ve = (el.dataset.ve || "").split(",").map((s) => s.trim());
      return ve.includes(eventName);
    }
    /**
     * Get modifier options for an event on an element.
     * Reads data-pd-{event}, data-sp-{event}, data-self-{event}, etc.
     * Per spec section 3.9.3, lines 1299-1365.
     */
    _getModifiers(el, eventName) {
      return {
        preventDefault: el.dataset[`pd${this._capitalize(eventName)}`] === "true",
        stopPropagation: el.dataset[`sp${this._capitalize(eventName)}`] === "true",
        self: el.dataset[`self${this._capitalize(eventName)}`] === "true",
        once: el.dataset[`once${this._capitalize(eventName)}`] === "true",
        passive: el.dataset[`passive${this._capitalize(eventName)}`] === "true",
        capture: el.dataset[`capture${this._capitalize(eventName)}`] === "true",
        debounce: parseInt(el.dataset[`debounce${this._capitalize(eventName)}`] || el.dataset.debounce || "0", 10),
        throttle: parseInt(el.dataset[`throttle${this._capitalize(eventName)}`] || el.dataset.throttle || "0", 10)
      };
    }
    /**
     * Capitalize first letter for dataset property access (click -> Click)
     */
    _capitalize(str) {
      return str.charAt(0).toUpperCase() + str.slice(1);
    }
    /**
     * Apply modifiers to an event.
     * Returns false if the event should not be processed.
     * Returns the modifiers object for additional checks by handlers.
     */
    _applyModifiers(event, el, eventName) {
      const mods = this._getModifiers(el, eventName);
      if (mods.self && event.target !== el) {
        return false;
      }
      if (mods.passive) {
        const originalPreventDefault = event.preventDefault.bind(event);
        event.preventDefault = () => {
          console.warn("[Vango] preventDefault() called on passive handler - ignored");
        };
        queueMicrotask(() => {
          event.preventDefault = originalPreventDefault;
        });
      } else if (mods.preventDefault) {
        event.preventDefault();
      }
      if (mods.stopPropagation) {
        event.stopPropagation();
      }
      if (mods.once) {
        const ve = (el.dataset.ve || "").split(",").map((s) => s.trim());
        const filtered = ve.filter((e) => e !== eventName);
        if (filtered.length > 0) {
          el.dataset.ve = filtered.join(",");
        } else {
          delete el.dataset.ve;
        }
      }
      return true;
    }
    /**
     * Get debounce delay for an event on an element.
     */
    _getDebounce(el, eventName) {
      const mods = this._getModifiers(el, eventName);
      if (eventName === "input" && mods.debounce === 0) {
        return 100;
      }
      return mods.debounce;
    }
    /**
     * Get throttle delay for an event on an element.
     */
    _getThrottle(el, eventName) {
      const mods = this._getModifiers(el, eventName);
      if (eventName === "scroll" && mods.throttle === 0) {
        return 100;
      }
      return mods.throttle;
    }
    /**
     * Find closest element with data-hid that has a specific event in data-ve.
     * Parses data-ve="click,input,change" format per spec Section 5.2.
     * This handles event bubbling through nested HID elements.
     */
    _findHidElementWithEvent(target, eventName) {
      if (!target || !target.closest) {
        return null;
      }
      let el = target.closest("[data-hid]");
      while (el) {
        if (this._hasEvent(el, eventName)) {
          return el;
        }
        const parent = el.parentElement;
        if (!parent)
          break;
        el = parent.closest("[data-hid]");
      }
      return null;
    }
    /**
     * Handle click event
     *
     * Per spec Section 5.2 (Interception Decision Table):
     * - MUST NOT intercept if defaultPrevented
     * - MUST NOT intercept right/middle click (button !== 0)
     * - MUST NOT intercept if modifier keys held
     * - MUST NOT intercept if WebSocket not connected
     * - For anchors: respect target, download, cross-origin
     */
    _handleClick(event) {
      if (event.defaultPrevented)
        return;
      if (event.button !== 0)
        return;
      if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey)
        return;
      if (!this.client.connected)
        return;
      const el = this._findHidElementWithEvent(event.target, "click");
      if (!el)
        return;
      const anchor = this._closest(event.target, "a[href]");
      if (anchor) {
        const target = anchor.getAttribute("target");
        if (target && target !== "_self")
          return;
        if (anchor.hasAttribute("download"))
          return;
        const href = anchor.getAttribute("href");
        if (href) {
          try {
            const url = new URL(href, location.href);
            if (url.origin !== location.origin)
              return;
          } catch (e) {
            return;
          }
        }
      }
      if (!this._applyModifiers(event, el, "click")) {
        return;
      }
      event.preventDefault();
      this.client.optimistic.applyOptimistic(el, "click");
      this.client.sendEvent(EventType.CLICK, el.dataset.hid);
    }
    /**
     * Handle double-click event
     */
    _handleDblClick(event) {
      const el = this._findHidElement(event.target);
      if (!el || !this._hasEvent(el, "dblclick"))
        return;
      if (!this._applyModifiers(event, el, "dblclick")) {
        return;
      }
      const mods = this._getModifiers(el, "dblclick");
      if (!mods.preventDefault && !mods.passive) {
        event.preventDefault();
      }
      this.client.sendEvent(EventType.DBLCLICK, el.dataset.hid);
    }
    /**
     * Handle input event (debounced)
     */
    _handleInput(event) {
      const el = this._findHidElement(event.target);
      if (!el || !this._hasEvent(el, "input"))
        return;
      if (!this._applyModifiers(event, el, "input")) {
        return;
      }
      const hid = el.dataset.hid;
      const debounceMs = this._getDebounce(el, "input");
      if (this.debounceTimers.has(hid)) {
        clearTimeout(this.debounceTimers.get(hid));
      }
      if (debounceMs === 0) {
        this.client.sendEvent(EventType.INPUT, hid, { value: el.value });
        return;
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
      if (!el || !this._hasEvent(el, "change"))
        return;
      if (!this._applyModifiers(event, el, "change")) {
        return;
      }
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
     *
     * Per spec Section 5.2 (Progressive Enhancement):
     * When WebSocket is unavailable, forms MUST fall back to normal HTTP submission.
     */
    _handleSubmit(event) {
      const form = this._closest(event.target, "form[data-hid]");
      if (!form || !this._hasEvent(form, "submit"))
        return;
      if (!this.client.connected)
        return;
      if (event.defaultPrevented)
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
      if (!el || !this._hasEvent(el, "focus"))
        return;
      if (!this._applyModifiers(event, el, "focus")) {
        return;
      }
      this.client.sendEvent(EventType.FOCUS, el.dataset.hid);
    }
    /**
     * Handle blur event
     */
    _handleBlur(event) {
      const el = this._findHidElement(event.target);
      if (!el || !this._hasEvent(el, "blur"))
        return;
      if (!this._applyModifiers(event, el, "blur")) {
        return;
      }
      this.client.sendEvent(EventType.BLUR, el.dataset.hid);
    }
    /**
     * Handle keydown event
     */
    _handleKeyDown(event) {
      const el = this._findHidElement(event.target);
      if (!el || !this._hasEvent(el, "keydown"))
        return;
      const keyFilter = el.dataset.keyFilter;
      if (keyFilter && !this._matchesKeyFilter(event, keyFilter)) {
        return;
      }
      if (!this._applyModifiers(event, el, "keydown")) {
        return;
      }
      this.client.sendEvent(EventType.KEYDOWN, el.dataset.hid, {
        key: event.key,
        code: event.code,
        ctrlKey: event.ctrlKey,
        shiftKey: event.shiftKey,
        altKey: event.altKey,
        metaKey: event.metaKey,
        repeat: event.repeat,
        location: event.location
      });
    }
    /**
     * Handle keyup event
     */
    _handleKeyUp(event) {
      const el = this._findHidElement(event.target);
      if (!el || !this._hasEvent(el, "keyup"))
        return;
      if (!this._applyModifiers(event, el, "keyup")) {
        return;
      }
      this.client.sendEvent(EventType.KEYUP, el.dataset.hid, {
        key: event.key,
        code: event.code,
        ctrlKey: event.ctrlKey,
        shiftKey: event.shiftKey,
        altKey: event.altKey,
        metaKey: event.metaKey,
        repeat: event.repeat,
        location: event.location
      });
    }
    /**
     * Handle mouseenter event
     */
    _handleMouseEnter(event) {
      const el = this._findHidElement(event.target);
      if (!el || !this._hasEvent(el, "mouseenter"))
        return;
      if (!this._applyModifiers(event, el, "mouseenter")) {
        return;
      }
      this.client.sendEvent(EventType.MOUSEENTER, el.dataset.hid);
    }
    /**
     * Handle mouseleave event
     */
    _handleMouseLeave(event) {
      const el = this._findHidElement(event.target);
      if (!el || !this._hasEvent(el, "mouseleave"))
        return;
      if (!this._applyModifiers(event, el, "mouseleave")) {
        return;
      }
      this.client.sendEvent(EventType.MOUSELEAVE, el.dataset.hid);
    }
    /**
     * Handle scroll event (throttled)
     */
    _handleScroll(event) {
      const el = this._findHidElement(event.target);
      if (!el || !this._hasEvent(el, "scroll"))
        return;
      if (!this._applyModifiers(event, el, "scroll")) {
        return;
      }
      const hid = el.dataset.hid;
      const throttleMs = this._getThrottle(el, "scroll");
      if (throttleMs > 0 && this.scrollThrottled.has(hid)) {
        return;
      }
      if (throttleMs > 0) {
        this.scrollThrottled.add(hid);
        setTimeout(() => {
          this.scrollThrottled.delete(hid);
        }, throttleMs);
      }
      this.client.sendEvent(EventType.SCROLL, hid, {
        scrollTop: el.scrollTop,
        scrollLeft: el.scrollLeft
      });
    }
    /**
     * Handle link click for client-side navigation.
     *
     * Per Navigation Contract Section 4.5 (Link Click Navigation):
     * 1. Do NOT update history immediately
     * 2. Send EventNavigate { path, replace }
     * 3. Wait for server response with NAV_* + DOM patches
     * 4. Apply patches (URL update + DOM update)
     *
     * Per Progressive Enhancement Contract Section 5.1:
     * A link click is intercepted ONLY when ALL conditions are true:
     * 1. Element has data-vango-link attribute (canonical marker)
     * 2. WebSocket connection is healthy
     * 3. Link is same-origin
     * 4. No modifier keys pressed (ctrl, meta, shift, alt)
     * 5. No target attribute (or target="_self")
     * 6. No download attribute
     *
     * Note: data-link is supported for backwards compatibility but deprecated.
     */
    _handleLinkClick(event) {
      const link = this._closest(event.target, "a[href]");
      if (!link)
        return;
      if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
        return;
      }
      if (link.hasAttribute("download")) {
        return;
      }
      const target = link.getAttribute("target");
      if (target && target !== "_self") {
        return;
      }
      const isVangoLink = link.hasAttribute("data-vango-link") || link.hasAttribute("data-link");
      if (!isVangoLink) {
        return;
      }
      const href = link.getAttribute("href");
      if (!href)
        return;
      if (href.startsWith("http://") || href.startsWith("https://") || href.startsWith("//")) {
        try {
          const url = new URL(href, window.location.origin);
          if (url.origin !== window.location.origin) {
            return;
          }
        } catch (e) {
          return;
        }
      }
      if (link.hasAttribute("data-external")) {
        return;
      }
      if (!this.client.connected) {
        return;
      }
      event.preventDefault();
      this.pendingNavPath = href;
      this.client.sendEvent(EventType.NAVIGATE, "nav", { path: href, replace: false });
    }
    /**
     * Handle browser back/forward navigation (popstate).
     *
     * Per Navigation Contract Section 4.6:
     * 1. Browser fires popstate event
     * 2. Client sends EventNavigate { path, replace: true }
     * 3. Server remounts and sends DOM patches
     * 4. Server MAY omit NAV_REPLACE since URL already changed
     *
     * Note: popstate means the URL has already changed in the browser,
     * so we use replace: true to avoid server sending NAV_* that would
     * duplicate the history entry.
     */
    _handlePopState(event) {
      if (!this.client.connected) {
        return;
      }
      const path = location.pathname + location.search;
      this.pendingNavPath = path;
      this.client.sendEvent(EventType.NAVIGATE, "nav", { path, replace: true });
    }
    /**
     * Handle prefetch on hover for links with data-prefetch attribute.
     *
     * Per Prefetch Contract Section 8.1:
     * Client sends EventType.CUSTOM (0xFF) with:
     * - Name: "prefetch"
     * - Data: JSON-encoded { "path": "/target/path" }
     *
     * Wire format:
     * [0xFF (CUSTOM)][name: "prefetch" (varint-len string)][data: JSON bytes (varint-len)]
     *
     * The server can use this to preload route data before the user clicks,
     * making navigation feel instant.
     */
    _handlePrefetch(event) {
      const link = this._closest(event.target, "a[data-prefetch][href]");
      if (!link)
        return;
      const href = link.getAttribute("href");
      if (!href)
        return;
      if (href.startsWith("http://") || href.startsWith("https://") || href.startsWith("//")) {
        try {
          const url = new URL(href, window.location.origin);
          if (url.origin !== window.location.origin) {
            return;
          }
        } catch (e) {
          return;
        }
      }
      if (this.prefetchedPaths.has(href)) {
        return;
      }
      if (!this.client.connected) {
        return;
      }
      this.prefetchedPaths.add(href);
      const jsonData = JSON.stringify({ path: href });
      const encoder = new TextEncoder();
      const dataBytes = encoder.encode(jsonData);
      this.client.sendEvent(EventType.CUSTOM, "prefetch", { name: "prefetch", data: dataBytes });
      if (this.client.options.debug) {
        console.log("[Vango] Prefetching:", href);
      }
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
     * Check if a patch type requires a target DOM node.
     * URL/NAV patches operate on browser history, not DOM elements.
     * Per spec Section 9.6.1: These must not trigger self-heal.
     */
    _requiresTargetNode(patchType) {
      switch (patchType) {
        case PatchType.URL_PUSH:
        case PatchType.URL_REPLACE:
        case PatchType.NAV_PUSH:
        case PatchType.NAV_REPLACE:
          return false;
        case PatchType.INSERT_NODE:
          return false;
        default:
          return true;
      }
    }
    /**
     * Trigger self-heal recovery per spec Section 9.6.1.
     * If navigation in progress: location.assign(pendingPath)
     * Else: location.reload()
     */
    _triggerSelfHeal() {
      var _a;
      const pendingPath = (_a = this.client.eventCapture) == null ? void 0 : _a.pendingNavPath;
      if (pendingPath) {
        console.log("[Vango] Self-heal: navigating to pending path:", pendingPath);
        location.assign(pendingPath);
      } else {
        console.log("[Vango] Self-heal: reloading page");
        location.reload();
      }
    }
    /**
     * Apply single patch
     */
    applyPatch(patch) {
      const el = this.client.getNode(patch.hid);
      if (!el && this._requiresTargetNode(patch.type)) {
        console.error("[Vango] Patch target not found (HID:", patch.hid, "type:", patch.type, ")");
        this._triggerSelfHeal();
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
        case PatchType.URL_PUSH:
        case PatchType.URL_REPLACE:
          if (this.client.urlManager) {
            this.client.urlManager.applyPatch(patch);
          }
          break;
        case PatchType.NAV_PUSH:
        case PatchType.NAV_REPLACE:
          this._applyNavPatch(patch);
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
      if (key.length > 2 && key.substring(0, 2).toLowerCase() === "on") {
        console.warn("[Vango] Blocked dangerous attribute:", key);
        return;
      }
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
        console.error("[Vango] INSERT_NODE parent not found (parentHID:", parentHid, ")");
        this._triggerSelfHeal();
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
        console.error("[Vango] MOVE_NODE parent not found (parentHID:", parentHid, ")");
        this._triggerSelfHeal();
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
      if (eventName === "vango:navigate") {
        this._handleNavigate(parsedDetail);
        return;
      }
      const target = el || document;
      const event = new CustomEvent(eventName, {
        detail: parsedDetail,
        bubbles: true,
        cancelable: true
      });
      target.dispatchEvent(event);
    }
    /**
     * Handle server-initiated navigation via NAV_PUSH/NAV_REPLACE patches.
     * This is the contract-compliant implementation that does NOT send
     * EventNavigate back to the server (server already rendered the new page).
     *
     * Per spec Section 4.2 (Navigation Contract):
     * 1. Receive NAV_* patch
     * 2. Update history via pushState/replaceState with provided path
     * 3. Apply subsequent DOM patches
     * 4. Do NOT send EventNavigate back to server
     */
    _applyNavPatch(patch) {
      const { path, type } = patch;
      if (!path || !path.startsWith("/")) {
        console.error("[Vango] Invalid navigation path (must start with /):", path);
        return;
      }
      if (path.includes("://") || path.startsWith("//")) {
        console.error("[Vango] Invalid navigation path (must be relative):", path);
        return;
      }
      if (type === PatchType.NAV_REPLACE) {
        history.replaceState(null, "", path);
      } else {
        history.pushState(null, "", path);
      }
      if (this.client.eventCapture) {
        this.client.eventCapture.pendingNavPath = null;
      }
      window.scrollTo({ top: 0, behavior: "instant" });
      if (this.client.options.debug) {
        console.log("[Vango] Navigation applied:", path, type === PatchType.NAV_REPLACE ? "(replace)" : "(push)");
      }
    }
    /**
     * Handle server-initiated navigation via dispatch patch (DEPRECATED).
     * This is the legacy mechanism using vango:navigate custom event.
     * New code should use NAV_PUSH/NAV_REPLACE patches instead.
     */
    _handleNavigate(data) {
      if (!data || !data.path) {
        if (this.client.options.debug) {
          console.warn("[Vango] Invalid navigate data:", data);
        }
        return;
      }
      const { path, replace, scroll } = data;
      if (replace) {
        history.replaceState(null, "", path);
      } else {
        history.pushState(null, "", path);
      }
      this.client.sendEvent(EventType.NAVIGATE, "nav", { path });
      if (scroll !== false) {
        window.scrollTo({ top: 0, behavior: "instant" });
      }
      if (this.client.options.debug) {
        console.log("[Vango] Navigated to:", path, { replace, scroll });
      }
    }
    // NOTE: _evalCode method has been REMOVED for security.
    // Executing arbitrary JS from server is an XSS/RCE risk.
    // Use client-side hooks or custom events for safe JS interop.
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
     * Apply optimistic update based on element data attributes.
     * Parses data-optimistic='{"class":"...","text":"...","attr":"...","value":"..."}' format
     * per spec Section 5.2.
     */
    applyOptimistic(el, eventType) {
      const hid = el.dataset.hid;
      if (!hid)
        return;
      const optimisticData = el.dataset.optimistic;
      if (!optimisticData)
        return;
      try {
        const config = JSON.parse(optimisticData);
        if (config.class) {
          this._applyClassOptimistic(el, hid, config.class);
        }
        if (config.text) {
          this._applyTextOptimistic(el, hid, config.text);
        }
        if (config.attr && config.value !== void 0) {
          this._applyAttrOptimistic(el, hid, config.attr, config.value);
        }
      } catch (e) {
        console.warn("[Vango] Invalid optimistic config:", e);
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
      if (attr.length > 2 && attr.substring(0, 2).toLowerCase() === "on") {
        console.warn("[Vango] Blocked dangerous optimistic attribute:", attr);
        return;
      }
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
      var _a;
      const item = this._findItem(e.target);
      if (!item)
        return;
      if (this.handle && (typeof ((_a = e.target) == null ? void 0 : _a.closest) !== "function" || !e.target.closest(this.handle)))
        return;
      e.preventDefault();
      this._startDrag(item, e.clientX, e.clientY);
    }
    _handleTouchStart(e) {
      var _a;
      const item = this._findItem(e.target);
      if (!item)
        return;
      if (this.handle && (typeof ((_a = e.target) == null ? void 0 : _a.closest) !== "function" || !e.target.closest(this.handle)))
        return;
      e.preventDefault();
      const touch = e.touches[0];
      this._startDrag(item, touch.clientX, touch.clientY);
    }
    _findItem(target) {
      let item = target;
      while (item && item.parentElement !== this.el) {
        item = item.parentElement;
      }
      return item;
    }
    _startDrag(item, x, y) {
      this.dragging = item;
      this.activeContainer = item.parentElement;
      this.initialContainer = this.activeContainer;
      this.startIndex = Array.from(this.activeContainer.children).indexOf(item);
      this.startY = y;
      this.startX = x;
      this.itemHeight = item.offsetHeight;
      const rect = item.getBoundingClientRect();
      this.ghostStartTop = rect.top;
      this.ghostStartLeft = rect.left;
      this.ghost = item.cloneNode(true);
      this.ghost.classList.add(this.ghostClass);
      this.ghost.style.position = "fixed";
      this.ghost.style.zIndex = "9999";
      this.ghost.style.width = `${item.offsetWidth}px`;
      this.ghost.style.height = `${item.offsetHeight}px`;
      this.ghost.style.left = `${this.ghostStartLeft}px`;
      this.ghost.style.top = `${this.ghostStartTop}px`;
      this.ghost.style.pointerEvents = "none";
      this.ghost.style.opacity = "0.8";
      this.ghost.style.transition = "none";
      document.body.appendChild(this.ghost);
      item.classList.add(this.dragClass);
      item.style.opacity = "0.4";
    }
    _handleMouseMove(e) {
      if (!this.dragging)
        return;
      e.preventDefault();
      this._updateDrag(e.clientX, e.clientY);
    }
    _handleTouchMove(e) {
      if (!this.dragging)
        return;
      e.preventDefault();
      const touch = e.touches[0];
      this._updateDrag(touch.clientX, touch.clientY);
    }
    _updateDrag(x, y) {
      const deltaY = y - this.startY;
      const deltaX = x - this.startX;
      if (this.ghost) {
        this.ghost.style.top = `${this.ghostStartTop + deltaY}px`;
        this.ghost.style.left = `${this.ghostStartLeft + deltaX}px`;
      }
      this.ghost.style.display = "none";
      const elUnderCursor = document.elementFromPoint(x, y);
      this.ghost.style.display = "";
      if (elUnderCursor) {
        const targetContainer = elUnderCursor.closest('[data-hook="Sortable"]');
        if (targetContainer && targetContainer !== this.activeContainer && targetContainer.dataset.hookConfig) {
          try {
            const targetConfig = JSON.parse(targetContainer.dataset.hookConfig);
            if (targetConfig.group === this.config.group) {
              this.activeContainer = targetContainer;
              this.activeContainer.appendChild(this.dragging);
            }
          } catch (e) {
          }
        }
      }
      const children = Array.from(this.activeContainer.children);
      const currentIndex = children.indexOf(this.dragging);
      for (let i = 0; i < children.length; i++) {
        if (i === currentIndex)
          continue;
        const child = children[i];
        const rect = child.getBoundingClientRect();
        const midpoint = rect.top + rect.height / 2;
        if (y < midpoint && i < currentIndex) {
          this.activeContainer.insertBefore(this.dragging, child);
          break;
        } else if (y > midpoint && i > currentIndex) {
          this.activeContainer.insertBefore(this.dragging, child.nextSibling);
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
      const endIndex = Array.from(this.activeContainer.children).indexOf(this.dragging);
      const id = this.dragging.dataset.id || this.dragging.dataset.hid;
      const targetContainerHid = this.activeContainer.dataset.hid;
      this.dragging.classList.remove(this.dragClass);
      this.dragging.style.opacity = "";
      if (this.ghost) {
        this.ghost.remove();
      }
      if (this.activeContainer !== this.initialContainer || endIndex !== this.startIndex) {
        this.pushEvent("reorder", {
          id,
          fromContainer: this.initialContainer.dataset.id || this.initialContainer.dataset.hid,
          toContainer: this.activeContainer.dataset.id || this.activeContainer.dataset.hid,
          fromIndex: this.startIndex,
          toIndex: endIndex
        });
      }
      this.dragging = null;
      this.ghost = null;
      this.activeContainer = null;
      this.initialContainer = null;
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

  // src/hooks/droppable.js
  var _DroppableHook = class _DroppableHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.config = config;
      this.pushEvent = pushEvent;
      this.hoverClass = config.hoverClass || "drag-over";
      this._bind();
    }
    updated(el, config, pushEvent) {
      this.config = config;
      this.pushEvent = pushEvent;
      this.hoverClass = config.hoverClass || "drag-over";
    }
    destroyed() {
      this._unbind();
    }
    _bind() {
      this._enter = (e) => {
        e.preventDefault();
        this.el.classList.add(this.hoverClass);
      };
      this._over = (e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
      };
      this._leave = (e) => {
        if (!this.el.contains(e.relatedTarget))
          this.el.classList.remove(this.hoverClass);
      };
      this._drop = (e) => {
        var _a, _b;
        e.preventDefault();
        this.el.classList.remove(this.hoverClass);
        const data = { x: e.clientX, y: e.clientY };
        if (window.__vango_dragging__) {
          const d = window.__vango_dragging__;
          data.id = d.dataset.id || d.dataset.hid;
          window.__vango_dragging__ = null;
        }
        const files = (_a = e.dataTransfer) == null ? void 0 : _a.files;
        if (files == null ? void 0 : files.length)
          data.fileNames = Array.from(files).map((f) => f.name);
        const text = (_b = e.dataTransfer) == null ? void 0 : _b.getData("text/plain");
        if (text)
          data.text = text;
        this.pushEvent("drop", data);
      };
      this.el.addEventListener("dragenter", this._enter);
      this.el.addEventListener("dragover", this._over);
      this.el.addEventListener("dragleave", this._leave);
      this.el.addEventListener("drop", this._drop);
    }
    _unbind() {
      this.el.removeEventListener("dragenter", this._enter);
      this.el.removeEventListener("dragover", this._over);
      this.el.removeEventListener("dragleave", this._leave);
      this.el.removeEventListener("drop", this._drop);
    }
  };
  __name(_DroppableHook, "DroppableHook");
  var DroppableHook = _DroppableHook;

  // src/hooks/resizable.js
  var _ResizableHook = class _ResizableHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.config = config;
      this.pushEvent = pushEvent;
      this.handles = (config.handles || "se").split(",").map((h) => h.trim());
      this.resizing = false;
      this._handleEls = [];
      this._createHandles();
      this._bind();
    }
    updated(el, config, pushEvent) {
      this.config = config;
      this.pushEvent = pushEvent;
    }
    destroyed() {
      this._unbind();
      this._handleEls.forEach((h) => h.remove());
    }
    _createHandles() {
      if (getComputedStyle(this.el).position === "static") {
        this.el.style.position = "relative";
      }
      const cursors = { n: "n", s: "s", e: "e", w: "w", ne: "ne", se: "se", sw: "sw", nw: "nw" };
      for (const h of this.handles) {
        if (!cursors[h])
          continue;
        const el = document.createElement("div");
        el.dataset.handle = h;
        el.style.cssText = `position:absolute;z-index:10;cursor:${h}-resize;` + this._pos(h);
        this.el.appendChild(el);
        this._handleEls.push(el);
      }
    }
    _pos(h) {
      const s = 8, hs = 4;
      const m = {
        n: `top:-${hs}px;left:25%;width:50%;height:${s}px`,
        s: `bottom:-${hs}px;left:25%;width:50%;height:${s}px`,
        e: `right:-${hs}px;top:25%;width:${s}px;height:50%`,
        w: `left:-${hs}px;top:25%;width:${s}px;height:50%`,
        ne: `top:-${hs}px;right:-${hs}px;width:${s}px;height:${s}px`,
        se: `bottom:-${hs}px;right:-${hs}px;width:${s}px;height:${s}px`,
        sw: `bottom:-${hs}px;left:-${hs}px;width:${s}px;height:${s}px`,
        nw: `top:-${hs}px;left:-${hs}px;width:${s}px;height:${s}px`
      };
      return m[h] || "";
    }
    _bind() {
      this._onDown = (e) => {
        e.preventDefault();
        this._start(e.target.dataset.handle, e.clientX, e.clientY);
      };
      this._onMove = (e) => {
        if (this.resizing) {
          e.preventDefault();
          this._update(e.clientX, e.clientY);
        }
      };
      this._onUp = () => {
        if (this.resizing)
          this._end();
      };
      this._handleEls.forEach((h) => h.addEventListener("mousedown", this._onDown));
      document.addEventListener("mousemove", this._onMove);
      document.addEventListener("mouseup", this._onUp);
    }
    _unbind() {
      this._handleEls.forEach((h) => h.removeEventListener("mousedown", this._onDown));
      document.removeEventListener("mousemove", this._onMove);
      document.removeEventListener("mouseup", this._onUp);
    }
    _start(handle, x, y) {
      this.resizing = true;
      this._h = handle;
      this._sx = x;
      this._sy = y;
      const r = this.el.getBoundingClientRect();
      this._sw = r.width;
      this._sh = r.height;
    }
    _update(x, y) {
      const dx = x - this._sx, dy = y - this._sy;
      let w = this._sw, h = this._sh;
      if (this._h.includes("e"))
        w = this._sw + dx;
      if (this._h.includes("w"))
        w = this._sw - dx;
      if (this._h.includes("s"))
        h = this._sh + dy;
      if (this._h.includes("n"))
        h = this._sh - dy;
      const c = this.config;
      w = Math.max(c.minWidth || 0, Math.min(c.maxWidth || Infinity, w));
      h = Math.max(c.minHeight || 0, Math.min(c.maxHeight || Infinity, h));
      this.el.style.width = w + "px";
      this.el.style.height = h + "px";
    }
    _end() {
      this.resizing = false;
      const r = this.el.getBoundingClientRect();
      this.pushEvent("resize", { width: Math.round(r.width), height: Math.round(r.height) });
    }
  };
  __name(_ResizableHook, "ResizableHook");
  var ResizableHook = _ResizableHook;

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

  // src/hooks/collapsible.js
  var _CollapsibleHook = class _CollapsibleHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.config = config;
      this.pushEvent = pushEvent;
      this.duration = config.duration || 200;
      this.isOpen = !!config.open;
      this.animating = false;
      if (!this.isOpen) {
        el.style.height = "0";
        el.style.overflow = "hidden";
      }
    }
    updated(el, config, pushEvent) {
      this.pushEvent = pushEvent;
      this.duration = config.duration || 200;
      const newOpen = !!config.open;
      if (newOpen !== this.isOpen) {
        newOpen ? this._expand() : this._collapse();
      }
    }
    destroyed() {
    }
    _expand() {
      if (this.animating || this.isOpen)
        return;
      this.animating = true;
      this.isOpen = true;
      const el = this.el;
      el.style.height = "auto";
      const h = el.scrollHeight;
      el.style.height = "0";
      el.style.overflow = "hidden";
      el.offsetHeight;
      el.style.transition = `height ${this.duration}ms ease`;
      el.style.height = h + "px";
      setTimeout(() => {
        el.style.transition = "";
        el.style.height = "auto";
        el.style.overflow = "";
        this.animating = false;
        this.pushEvent("toggle", { open: true });
      }, this.duration);
    }
    _collapse() {
      if (this.animating || !this.isOpen)
        return;
      this.animating = true;
      this.isOpen = false;
      const el = this.el;
      el.style.height = el.scrollHeight + "px";
      el.style.overflow = "hidden";
      el.offsetHeight;
      el.style.transition = `height ${this.duration}ms ease`;
      el.style.height = "0";
      setTimeout(() => {
        el.style.transition = "";
        this.animating = false;
        this.pushEvent("toggle", { open: false });
      }, this.duration);
    }
  };
  __name(_CollapsibleHook, "CollapsibleHook");
  var CollapsibleHook = _CollapsibleHook;

  // src/hooks/focustrap.js
  var _FocusTrapHook = class _FocusTrapHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.pushEvent = pushEvent;
      this.active = config.active !== false;
      this.restoreFocusTo = document.activeElement;
      this._onKeyDown = this._handleKeyDown.bind(this);
      this.el.addEventListener("keydown", this._onKeyDown);
      if (this.active) {
        this._focusFirst();
      }
    }
    updated(el, config, pushEvent) {
      this.active = config.active !== false;
      this.pushEvent = pushEvent;
    }
    destroyed(el) {
      this.el.removeEventListener("keydown", this._onKeyDown);
      if (this.restoreFocusTo && typeof this.restoreFocusTo.focus === "function") {
        this.restoreFocusTo.focus();
      }
    }
    _handleKeyDown(e) {
      if (!this.active || e.key !== "Tab")
        return;
      const focusable = this._getFocusableElements();
      if (focusable.length === 0) {
        e.preventDefault();
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
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
        this.el.setAttribute("tabindex", "-1");
        this.el.focus();
      }
    }
    _getFocusableElements() {
      return this.el.querySelectorAll(
        'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
      );
    }
  };
  __name(_FocusTrapHook, "FocusTrapHook");
  var FocusTrapHook = _FocusTrapHook;

  // src/hooks/portal.js
  var _PortalHook = class _PortalHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.pushEvent = pushEvent;
      this.target = config.target || "body";
      this.placeholder = document.createComment("portal-placeholder");
      if (el.parentNode) {
        el.parentNode.insertBefore(this.placeholder, el);
      }
      let portalRoot = document.getElementById("vango-portal-root");
      if (!portalRoot) {
        portalRoot = document.createElement("div");
        portalRoot.id = "vango-portal-root";
        portalRoot.style.cssText = "position: relative; z-index: 9999;";
        document.body.appendChild(portalRoot);
      }
      portalRoot.appendChild(el);
    }
    updated(el, config, pushEvent) {
      this.pushEvent = pushEvent;
    }
    destroyed(el) {
      if (this.placeholder && this.placeholder.parentNode) {
        this.placeholder.parentNode.insertBefore(this.el, this.placeholder);
        this.placeholder.remove();
      }
    }
  };
  __name(_PortalHook, "PortalHook");
  var PortalHook = _PortalHook;
  function ensurePortalRoot() {
    if (typeof document === "undefined")
      return;
    let portalRoot = document.getElementById("vango-portal-root");
    if (!portalRoot) {
      portalRoot = document.createElement("div");
      portalRoot.id = "vango-portal-root";
      portalRoot.style.cssText = "position: relative; z-index: 9999;";
      document.body.appendChild(portalRoot);
    }
    return portalRoot;
  }
  __name(ensurePortalRoot, "ensurePortalRoot");

  // src/hooks/dialog.js
  var _DialogHook = class _DialogHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.config = config;
      this.pushEvent = pushEvent;
      this.previousFocus = document.activeElement;
      this._onKeyDown = this._handleKeyDown.bind(this);
      this._onClick = this._handleClick.bind(this);
      document.addEventListener("keydown", this._onKeyDown);
      if (config.closeOnOutside !== false) {
        setTimeout(() => {
          document.addEventListener("click", this._onClick);
        }, 0);
      }
      this._focusFirst();
      this._originalOverflow = document.body.style.overflow;
      document.body.style.overflow = "hidden";
    }
    updated(el, config, pushEvent) {
      this.config = config;
      this.pushEvent = pushEvent;
    }
    destroyed(el) {
      document.removeEventListener("keydown", this._onKeyDown);
      document.removeEventListener("click", this._onClick);
      document.body.style.overflow = this._originalOverflow || "";
      if (this.previousFocus && typeof this.previousFocus.focus === "function") {
        this.previousFocus.focus();
      }
    }
    _handleKeyDown(e) {
      if (e.key === "Escape" && this.config.closeOnEscape !== false) {
        e.preventDefault();
        this.pushEvent("close", {});
        return;
      }
      if (e.key === "Tab") {
        this._trapFocus(e);
      }
    }
    _handleClick(e) {
      if (!this.el.contains(e.target)) {
        this.pushEvent("close", {});
      }
    }
    _focusFirst() {
      if (this.config.initialFocus) {
        const initial = this.el.querySelector(this.config.initialFocus);
        if (initial) {
          initial.focus();
          return;
        }
      }
      const focusable = this._getFocusableElements();
      if (focusable.length > 0) {
        focusable[0].focus();
      } else {
        this.el.setAttribute("tabindex", "-1");
        this.el.focus();
      }
    }
    _trapFocus(e) {
      const focusable = this._getFocusableElements();
      if (focusable.length === 0) {
        e.preventDefault();
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }
    _getFocusableElements() {
      return this.el.querySelectorAll(
        'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
      );
    }
  };
  __name(_DialogHook, "DialogHook");
  var DialogHook = _DialogHook;

  // src/hooks/popover.js
  var _PopoverHook = class _PopoverHook {
    mounted(el, config, pushEvent) {
      this.el = el;
      this.config = config;
      this.pushEvent = pushEvent;
      this.trigger = el.querySelector("[data-popover-trigger]");
      this.content = el.querySelector("[data-popover-content]");
      if (!this.content) {
        this.content = el;
      }
      this._onKeyDown = this._handleKeyDown.bind(this);
      this._onClick = this._handleClick.bind(this);
      this._onScroll = this._handleScroll.bind(this);
      this._onResize = this._handleResize.bind(this);
      document.addEventListener("keydown", this._onKeyDown);
      if (config.closeOnOutside !== false) {
        setTimeout(() => {
          document.addEventListener("click", this._onClick);
        }, 0);
      }
      window.addEventListener("scroll", this._onScroll, true);
      window.addEventListener("resize", this._onResize);
      this._position();
    }
    updated(el, config, pushEvent) {
      this.config = config;
      this.pushEvent = pushEvent;
      this._position();
    }
    destroyed(el) {
      document.removeEventListener("keydown", this._onKeyDown);
      document.removeEventListener("click", this._onClick);
      window.removeEventListener("scroll", this._onScroll, true);
      window.removeEventListener("resize", this._onResize);
    }
    _handleKeyDown(e) {
      if (e.key === "Escape" && this.config.closeOnEscape !== false) {
        e.preventDefault();
        this.pushEvent("close", {});
      }
    }
    _handleClick(e) {
      if (!this.el.contains(e.target)) {
        this.pushEvent("close", {});
      }
    }
    _handleScroll() {
      this._position();
    }
    _handleResize() {
      this._position();
    }
    _position() {
      if (!this.trigger || !this.content)
        return;
      const triggerRect = this.trigger.getBoundingClientRect();
      const contentRect = this.content.getBoundingClientRect();
      const offset = this.config.offset || 4;
      let side = this.config.side || "bottom";
      let align = this.config.align || "center";
      let top, left;
      switch (side) {
        case "top":
          top = triggerRect.top - contentRect.height - offset;
          if (top < 0) {
            top = triggerRect.bottom + offset;
            side = "bottom";
          }
          break;
        case "bottom":
          top = triggerRect.bottom + offset;
          if (top + contentRect.height > window.innerHeight) {
            top = triggerRect.top - contentRect.height - offset;
            side = "top";
          }
          break;
        case "left":
          left = triggerRect.left - contentRect.width - offset;
          if (left < 0) {
            left = triggerRect.right + offset;
            side = "right";
          }
          break;
        case "right":
          left = triggerRect.right + offset;
          if (left + contentRect.width > window.innerWidth) {
            left = triggerRect.left - contentRect.width - offset;
            side = "left";
          }
          break;
      }
      if (side === "top" || side === "bottom") {
        switch (align) {
          case "start":
            left = triggerRect.left;
            break;
          case "center":
            left = triggerRect.left + (triggerRect.width - contentRect.width) / 2;
            break;
          case "end":
            left = triggerRect.right - contentRect.width;
            break;
        }
        if (left < 0)
          left = 0;
        if (left + contentRect.width > window.innerWidth) {
          left = window.innerWidth - contentRect.width;
        }
      }
      if (side === "left" || side === "right") {
        switch (align) {
          case "start":
            top = triggerRect.top;
            break;
          case "center":
            top = triggerRect.top + (triggerRect.height - contentRect.height) / 2;
            break;
          case "end":
            top = triggerRect.bottom - contentRect.height;
            break;
        }
        if (top < 0)
          top = 0;
        if (top + contentRect.height > window.innerHeight) {
          top = window.innerHeight - contentRect.height;
        }
      }
      this.content.style.position = "fixed";
      this.content.style.top = `${top}px`;
      this.content.style.left = `${left}px`;
      this.content.dataset.side = side;
    }
  };
  __name(_PopoverHook, "PopoverHook");
  var PopoverHook = _PopoverHook;

  // src/hooks/manager.js
  var _HookManager = class _HookManager {
    constructor(client) {
      this.client = client;
      this.instances = /* @__PURE__ */ new Map();
      this.pendingReverts = /* @__PURE__ */ new Map();
      this.hooks = {
        "Sortable": SortableHook,
        "Draggable": DraggableHook,
        "Droppable": DroppableHook,
        "Resizable": ResizableHook,
        "Tooltip": TooltipHook,
        "Dropdown": DropdownHook,
        "Collapsible": CollapsibleHook,
        // VangoUI helper hooks
        "FocusTrap": FocusTrapHook,
        "Portal": PortalHook,
        "Dialog": DialogHook,
        "Popover": PopoverHook
      };
      document.addEventListener("vango:hook-revert", (e) => {
        var _a;
        const hid = (_a = e.detail) == null ? void 0 : _a.hid;
        if (hid && this.pendingReverts.has(hid)) {
          const revertFn = this.pendingReverts.get(hid);
          if (typeof revertFn === "function") {
            revertFn();
          }
          this.pendingReverts.delete(hid);
        }
      });
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
      const pushEvent = /* @__PURE__ */ __name((eventName, data = {}, revertFn = null) => {
        if (typeof revertFn === "function") {
          this.pendingReverts.set(hid, revertFn);
        }
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
        const pushEvent = /* @__PURE__ */ __name((eventName, data = {}, revertFn = null) => {
          if (typeof revertFn === "function") {
            this.pendingReverts.set(hid, revertFn);
          }
          this.client.sendHookEvent(hid, eventName, data);
        }, "pushEvent");
        entry.instance.updated(entry.el, config, pushEvent);
      }
    }
  };
  __name(_HookManager, "HookManager");
  var HookManager = _HookManager;

  // src/connection.js
  var ConnectionState = {
    CONNECTING: "connecting",
    CONNECTED: "connected",
    RECONNECTING: "reconnecting",
    DISCONNECTED: "disconnected"
  };
  var CSS_CLASSES = {
    [ConnectionState.CONNECTING]: "vango-connecting",
    [ConnectionState.CONNECTED]: "vango-connected",
    [ConnectionState.RECONNECTING]: "vango-reconnecting",
    [ConnectionState.DISCONNECTED]: "vango-disconnected"
  };
  var _ConnectionManager = class _ConnectionManager {
    constructor(options = {}) {
      this.options = {
        // Toast settings
        toastOnReconnect: options.toastOnReconnect || false,
        toastMessage: options.toastMessage || "Connection restored",
        toastDuration: options.toastDuration || 3e3,
        // Reconnection settings
        maxRetries: options.maxRetries || 10,
        baseDelay: options.baseDelay || 1e3,
        maxDelay: options.maxDelay || 3e4,
        // Debug mode
        debug: options.debug || false,
        ...options
      };
      this.state = ConnectionState.CONNECTING;
      this.retryCount = 0;
      this.previousState = null;
      this._updateClasses();
    }
    /**
     * Set connection state and update UI
     */
    setState(newState) {
      if (this.state === newState)
        return;
      this.previousState = this.state;
      this.state = newState;
      this._updateClasses();
      this._dispatchEvent(newState, this.previousState);
      if (this.options.toastOnReconnect && newState === ConnectionState.CONNECTED && this.previousState !== ConnectionState.CONNECTED) {
        this.showToast(this.options.toastMessage);
      }
      if (this.options.debug) {
        console.log(`[Vango] Connection state: ${this.previousState} -> ${newState}`);
      }
    }
    /**
     * Update CSS classes on document.documentElement (<html>)
     */
    _updateClasses() {
      const root = document.documentElement;
      Object.values(CSS_CLASSES).forEach((cls) => {
        root.classList.remove(cls);
      });
      const currentClass = CSS_CLASSES[this.state];
      if (currentClass) {
        root.classList.add(currentClass);
      }
    }
    /**
     * Dispatch custom event for connection state changes
     */
    _dispatchEvent(state, previousState) {
      const event = new CustomEvent("vango:connection", {
        detail: { state, previousState },
        bubbles: true
      });
      document.dispatchEvent(event);
    }
    /**
     * Handle connection established
     */
    onConnect() {
      this.retryCount = 0;
      this.setState(ConnectionState.CONNECTED);
    }
    /**
     * Handle connection lost
     */
    onDisconnect() {
      if (this.state === ConnectionState.CONNECTED) {
        this.setState(ConnectionState.RECONNECTING);
      }
    }
    /**
     * Handle failed reconnection attempt
     */
    onReconnectFailed() {
      this.retryCount++;
      if (this.retryCount >= this.options.maxRetries) {
        this.setState(ConnectionState.DISCONNECTED);
        return false;
      }
      return true;
    }
    /**
     * Calculate next retry delay with exponential backoff
     */
    getNextRetryDelay() {
      const delay = Math.min(
        this.options.baseDelay * Math.pow(2, this.retryCount),
        this.options.maxDelay
      );
      return delay;
    }
    /**
     * Get current state
     */
    getState() {
      return this.state;
    }
    /**
     * Check if currently connected
     */
    isConnected() {
      return this.state === ConnectionState.CONNECTED;
    }
    /**
     * Check if disconnected (gave up reconnecting)
     */
    isDisconnected() {
      return this.state === ConnectionState.DISCONNECTED;
    }
    /**
     * Reset state (e.g., for manual retry)
     */
    reset() {
      this.retryCount = 0;
      this.setState(ConnectionState.CONNECTED);
    }
    /**
     * Show a toast notification
     */
    showToast(message, duration = this.options.toastDuration) {
      if (typeof window.__VANGO_TOAST__ === "function") {
        window.__VANGO_TOAST__(message, duration);
        return;
      }
      const toast = document.createElement("div");
      toast.className = "vango-toast";
      toast.textContent = message;
      toast.setAttribute("role", "alert");
      toast.setAttribute("aria-live", "polite");
      Object.assign(toast.style, {
        position: "fixed",
        bottom: "20px",
        left: "50%",
        transform: "translateX(-50%)",
        padding: "12px 24px",
        backgroundColor: "hsl(var(--primary, 220 13% 18%))",
        color: "hsl(var(--primary-foreground, 0 0% 100%))",
        borderRadius: "var(--radius, 6px)",
        boxShadow: "0 4px 12px rgba(0, 0, 0, 0.15)",
        zIndex: "10001",
        opacity: "0",
        transition: "opacity 0.3s ease",
        fontFamily: "system-ui, -apple-system, sans-serif",
        fontSize: "14px"
      });
      document.body.appendChild(toast);
      requestAnimationFrame(() => {
        toast.style.opacity = "1";
      });
      setTimeout(() => {
        toast.style.opacity = "0";
        setTimeout(() => {
          if (toast.parentNode) {
            toast.parentNode.removeChild(toast);
          }
        }, 300);
      }, duration);
    }
  };
  __name(_ConnectionManager, "ConnectionManager");
  var ConnectionManager = _ConnectionManager;
  function injectDefaultStyles() {
    if (document.getElementById("vango-connection-styles")) {
      return;
    }
    const style = document.createElement("style");
    style.id = "vango-connection-styles";
    style.textContent = `
        /* Vango Connection State Styles */

        /* Connecting indicator - pulsing bar at top */
        html.vango-connecting::after {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            height: 3px;
            background: linear-gradient(90deg, transparent, var(--vango-primary, #3b82f6), transparent);
            animation: vango-connect-pulse 1.5s ease-in-out infinite;
            z-index: 9999;
        }

        @keyframes vango-connect-pulse {
            0%, 100% { opacity: 0.3; }
            50% { opacity: 1; }
        }

        /* Reconnecting indicator - pulsing bar at top */
        html.vango-reconnecting::after {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            height: 3px;
            background: linear-gradient(90deg, transparent, var(--vango-primary, #3b82f6), transparent);
            animation: vango-reconnect-pulse 1.5s ease-in-out infinite;
            z-index: 9999;
        }

        @keyframes vango-reconnect-pulse {
            0%, 100% { opacity: 0.3; }
            50% { opacity: 1; }
        }

        /* Disconnected state - slight grayscale and overlay */
        html.vango-disconnected {
            filter: grayscale(0.3);
        }

        html.vango-disconnected::before {
            content: 'Connection lost. Click to retry...';
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: hsl(var(--background, 0 0% 100%));
            border: 1px solid hsl(var(--border, 0 0% 90%));
            padding: 16px 32px;
            border-radius: var(--radius, 6px);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            z-index: 10000;
            font-family: system-ui, -apple-system, sans-serif;
            font-size: 14px;
            color: hsl(var(--foreground, 0 0% 9%));
            cursor: pointer;
        }

        /* Toast notifications */
        .vango-toast {
            animation: vango-toast-slide-up 0.3s ease-out;
        }

        @keyframes vango-toast-slide-up {
            from {
                transform: translateX(-50%) translateY(20px);
                opacity: 0;
            }
            to {
                transform: translateX(-50%) translateY(0);
                opacity: 1;
            }
        }
    `;
    document.head.appendChild(style);
  }
  __name(injectDefaultStyles, "injectDefaultStyles");

  // src/url.js
  var _URLManager = class _URLManager {
    constructor(client, options = {}) {
      this.client = client;
      this.options = {
        debug: options.debug || false,
        ...options
      };
      this.pending = /* @__PURE__ */ new Map();
    }
    /**
     * Apply a URL patch
     * @param {Object} patch - The URL patch with op and params
     */
    applyPatch(patch) {
      const { op, params } = patch;
      if (!params || typeof params !== "object") {
        if (this.options.debug) {
          console.warn("[Vango URL] Invalid params:", params);
        }
        return;
      }
      const url = new URL(window.location);
      for (const [key, value] of Object.entries(params)) {
        if (value === "" || value === null || value === void 0) {
          url.searchParams.delete(key);
        } else {
          url.searchParams.set(key, value);
        }
      }
      const isPush = op === 48;
      if (this.options.debug) {
        console.log("[Vango URL]", isPush ? "Push" : "Replace", url.toString());
      }
      if (isPush) {
        history.pushState(null, "", url.toString());
      } else {
        history.replaceState(null, "", url.toString());
      }
      this._dispatchEvent(params, isPush);
    }
    /**
     * Dispatch custom event for URL changes
     */
    _dispatchEvent(params, isPush) {
      const event = new CustomEvent("vango:url", {
        detail: {
          params,
          mode: isPush ? "push" : "replace",
          url: window.location.href
        },
        bubbles: true
      });
      document.dispatchEvent(event);
    }
    /**
     * Get current query parameters as an object
     */
    getParams() {
      const params = {};
      const searchParams = new URLSearchParams(window.location.search);
      for (const [key, value] of searchParams) {
        params[key] = value;
      }
      return params;
    }
    /**
     * Get a specific query parameter
     */
    getParam(key) {
      const searchParams = new URLSearchParams(window.location.search);
      return searchParams.get(key);
    }
    /**
     * Check if a query parameter exists
     */
    hasParam(key) {
      const searchParams = new URLSearchParams(window.location.search);
      return searchParams.has(key);
    }
  };
  __name(_URLManager, "URLManager");
  var URLManager = _URLManager;

  // src/prefs.js
  var MergeStrategy = {
    DB_WINS: 0,
    // Server value wins
    LOCAL_WINS: 1,
    // Local value wins
    PROMPT: 2,
    // Ask user to choose
    LWW: 3
    // Last-write-wins by timestamp
  };
  var _PrefManager = class _PrefManager {
    constructor(client, options = {}) {
      this.client = client;
      this.options = {
        channelName: "vango:prefs",
        storagePrefix: "vango_pref_",
        debug: options.debug || false,
        ...options
      };
      this.prefs = /* @__PURE__ */ new Map();
      this.channel = null;
      if (typeof BroadcastChannel !== "undefined") {
        this.channel = new BroadcastChannel(this.options.channelName);
        this.channel.onmessage = (event) => this._handleBroadcast(event);
      }
      if (typeof window !== "undefined") {
        window.addEventListener("storage", (event) => this._handleStorageEvent(event));
      }
    }
    /**
     * Register a preference
     * @param {string} key - Unique preference key
     * @param {*} defaultValue - Default value
     * @param {Object} options - Configuration options
     * @returns {Pref} Preference instance
     */
    register(key, defaultValue, options = {}) {
      const pref = new Pref(this, key, defaultValue, options);
      this.prefs.set(key, pref);
      pref._loadFromStorage();
      return pref;
    }
    /**
     * Get a registered preference
     */
    get(key) {
      return this.prefs.get(key);
    }
    /**
     * Broadcast a preference change to other tabs
     */
    broadcast(key, value, updatedAt) {
      if (this.channel) {
        this.channel.postMessage({
          type: "update",
          key,
          value,
          updatedAt: updatedAt.toISOString()
        });
      }
    }
    /**
     * Handle broadcast message from another tab
     */
    _handleBroadcast(event) {
      const { type, key, value, updatedAt } = event.data;
      if (type === "update") {
        const pref = this.prefs.get(key);
        if (pref) {
          pref._setFromRemote(value, new Date(updatedAt));
        }
      }
      if (this.options.debug) {
        console.log("[Vango Prefs] Broadcast received:", event.data);
      }
    }
    /**
     * Handle storage event (fallback for cross-tab sync)
     */
    _handleStorageEvent(event) {
      if (!event.key || !event.key.startsWith(this.options.storagePrefix)) {
        return;
      }
      const prefKey = event.key.slice(this.options.storagePrefix.length);
      const pref = this.prefs.get(prefKey);
      if (pref && event.newValue) {
        try {
          const data = JSON.parse(event.newValue);
          pref._setFromRemote(data.value, new Date(data.updatedAt));
        } catch (e) {
          if (this.options.debug) {
            console.warn("[Vango Prefs] Failed to parse storage event:", e);
          }
        }
      }
    }
    /**
     * Save to LocalStorage
     */
    saveToStorage(key, value, updatedAt) {
      if (typeof localStorage === "undefined")
        return;
      const storageKey = this.options.storagePrefix + key;
      const data = {
        value,
        updatedAt: updatedAt.toISOString()
      };
      try {
        localStorage.setItem(storageKey, JSON.stringify(data));
      } catch (e) {
        if (this.options.debug) {
          console.warn("[Vango Prefs] Failed to save to storage:", e);
        }
      }
    }
    /**
     * Load from LocalStorage
     */
    loadFromStorage(key) {
      if (typeof localStorage === "undefined")
        return null;
      const storageKey = this.options.storagePrefix + key;
      try {
        const data = localStorage.getItem(storageKey);
        if (data) {
          const parsed = JSON.parse(data);
          return {
            value: parsed.value,
            updatedAt: new Date(parsed.updatedAt)
          };
        }
      } catch (e) {
        if (this.options.debug) {
          console.warn("[Vango Prefs] Failed to load from storage:", e);
        }
      }
      return null;
    }
    /**
     * Remove from LocalStorage
     */
    removeFromStorage(key) {
      if (typeof localStorage === "undefined")
        return;
      const storageKey = this.options.storagePrefix + key;
      try {
        localStorage.removeItem(storageKey);
      } catch (e) {
      }
    }
    /**
     * Sync all preferences with server
     * Called when user logs in
     */
    async syncWithServer(serverPrefs) {
      for (const [key, serverData] of Object.entries(serverPrefs)) {
        const pref = this.prefs.get(key);
        if (pref) {
          pref._mergeWithServer(serverData.value, new Date(serverData.updatedAt));
        }
      }
    }
    /**
     * Get all preferences as an object for server sync
     */
    getAllForSync() {
      const result = {};
      for (const [key, pref] of this.prefs) {
        if (pref.options.syncToServer !== false) {
          result[key] = {
            value: pref.value,
            updatedAt: pref.updatedAt.toISOString()
          };
        }
      }
      return result;
    }
    /**
     * Cleanup
     */
    destroy() {
      if (this.channel) {
        this.channel.close();
        this.channel = null;
      }
    }
  };
  __name(_PrefManager, "PrefManager");
  var PrefManager = _PrefManager;
  var _Pref = class _Pref {
    constructor(manager, key, defaultValue, options = {}) {
      this.manager = manager;
      this.key = key;
      this.defaultValue = defaultValue;
      this.value = defaultValue;
      this.updatedAt = /* @__PURE__ */ new Date();
      this.options = {
        mergeStrategy: MergeStrategy.LWW,
        persistLocal: true,
        syncToServer: true,
        onConflict: null,
        // Custom conflict handler
        onChange: null,
        // Called when value changes
        ...options
      };
      this.subscribers = /* @__PURE__ */ new Set();
    }
    /**
     * Get the current value
     */
    get() {
      return this.value;
    }
    /**
     * Set a new value
     */
    set(value) {
      if (this._isEqual(this.value, value)) {
        return;
      }
      const oldValue = this.value;
      this.value = value;
      this.updatedAt = /* @__PURE__ */ new Date();
      if (this.options.persistLocal) {
        this.manager.saveToStorage(this.key, value, this.updatedAt);
      }
      this.manager.broadcast(this.key, value, this.updatedAt);
      if (this.options.syncToServer && this.manager.client) {
        this._syncToServer();
      }
      this._notifySubscribers(value, oldValue);
      if (this.options.onChange) {
        this.options.onChange(value, oldValue);
      }
    }
    /**
     * Reset to default value
     */
    reset() {
      this.set(this.defaultValue);
    }
    /**
     * Subscribe to value changes
     * @param {Function} callback - Called with (newValue, oldValue)
     * @returns {Function} Unsubscribe function
     */
    subscribe(callback) {
      this.subscribers.add(callback);
      return () => this.subscribers.delete(callback);
    }
    /**
     * Load initial value from storage
     */
    _loadFromStorage() {
      const stored = this.manager.loadFromStorage(this.key);
      if (stored) {
        this.value = stored.value;
        this.updatedAt = stored.updatedAt;
      }
    }
    /**
     * Set value from remote source (another tab)
     */
    _setFromRemote(value, remoteUpdatedAt) {
      const resolved = this._resolveConflict(this.value, value, this.updatedAt, remoteUpdatedAt);
      if (!this._isEqual(this.value, resolved)) {
        const oldValue = this.value;
        this.value = resolved;
        if (remoteUpdatedAt > this.updatedAt) {
          this.updatedAt = remoteUpdatedAt;
        }
        if (this.options.persistLocal) {
          this.manager.saveToStorage(this.key, this.value, this.updatedAt);
        }
        this._notifySubscribers(this.value, oldValue);
        if (this.options.onChange) {
          this.options.onChange(this.value, oldValue);
        }
      }
    }
    /**
     * Merge with server value (called on login)
     */
    _mergeWithServer(serverValue, serverUpdatedAt) {
      const resolved = this._resolveConflict(this.value, serverValue, this.updatedAt, serverUpdatedAt);
      if (!this._isEqual(this.value, resolved)) {
        const oldValue = this.value;
        this.value = resolved;
        if (serverUpdatedAt > this.updatedAt) {
          this.updatedAt = serverUpdatedAt;
        }
        if (this.options.persistLocal) {
          this.manager.saveToStorage(this.key, this.value, this.updatedAt);
        }
        this._notifySubscribers(this.value, oldValue);
        if (this.options.onChange) {
          this.options.onChange(this.value, oldValue);
        }
      }
    }
    /**
     * Resolve conflict between local and remote values
     */
    _resolveConflict(local, remote, localTime, remoteTime) {
      if (this.options.onConflict) {
        return this.options.onConflict(local, remote, localTime, remoteTime);
      }
      switch (this.options.mergeStrategy) {
        case MergeStrategy.DB_WINS:
          return remote;
        case MergeStrategy.LOCAL_WINS:
          return local;
        case MergeStrategy.LWW:
          return remoteTime > localTime ? remote : local;
        case MergeStrategy.PROMPT:
          return remoteTime > localTime ? remote : local;
        default:
          return local;
      }
    }
    /**
     * Sync preference to server
     */
    _syncToServer() {
      if (!this.manager.client || !this.manager.client.connected) {
        return;
      }
      this.manager.client.sendEvent(
        16,
        // EventType.PREF (custom event type for preferences)
        "",
        // No HID needed
        {
          key: this.key,
          value: this.value,
          updatedAt: this.updatedAt.toISOString()
        }
      );
    }
    /**
     * Notify all subscribers of value change
     */
    _notifySubscribers(newValue, oldValue) {
      for (const callback of this.subscribers) {
        try {
          callback(newValue, oldValue);
        } catch (e) {
          console.error("[Vango Prefs] Subscriber error:", e);
        }
      }
    }
    /**
     * Check if two values are equal
     */
    _isEqual(a, b) {
      if (a === b)
        return true;
      if (typeof a !== typeof b)
        return false;
      if (typeof a === "object" && a !== null && b !== null) {
        return JSON.stringify(a) === JSON.stringify(b);
      }
      return false;
    }
  };
  __name(_Pref, "Pref");
  var Pref = _Pref;

  // src/index.js
  var FrameType = {
    HANDSHAKE: 0,
    EVENT: 1,
    PATCHES: 2,
    CONTROL: 3,
    ACK: 4,
    ERROR: 5
  };
  var ControlType = {
    PING: 1,
    PONG: 2,
    RESYNC_REQUEST: 16,
    // Client -> Server: request missed patches
    RESYNC_PATCHES: 17,
    // Server -> Client: replay missed patches (not used with frame replay)
    RESYNC_FULL: 18,
    // Server -> Client: full HTML replacement
    CLOSE: 32
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
      this.patchSeq = 0;
      this.expectedPatchSeq = 1;
      this.pendingResync = false;
      this.wsManager = new WebSocketManager(this, this.options);
      this.patchApplier = new PatchApplier(this);
      this.eventCapture = new EventCapture(this);
      this.optimistic = new OptimisticUpdates(this);
      this.hooks = new HookManager(this);
      this.connection = new ConnectionManager({
        toastOnReconnect: options.toastOnReconnect || window.__VANGO_TOAST_ON_RECONNECT__,
        toastMessage: options.toastMessage || "Connection restored",
        maxRetries: options.maxRetries || 10,
        baseDelay: options.reconnectInterval || 1e3,
        maxDelay: options.reconnectMaxInterval || 3e4,
        debug: options.debug
      });
      this.urlManager = new URLManager(this, { debug: options.debug });
      this.prefs = new PrefManager(this, { debug: options.debug });
      this.onConnect = options.onConnect || (() => {
      });
      this.onDisconnect = options.onDisconnect || (() => {
      });
      this.onError = options.onError || ((err) => console.error("[Vango]", err));
      ensurePortalRoot();
      injectDefaultStyles();
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
      const path = encodeURIComponent(location.pathname + location.search);
      return `${protocol}//${location.host}/_vango/live?path=${path}`;
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
      if (buffer.length < 4)
        return;
      const frameType = buffer[0];
      const length = buffer[2] << 8 | buffer[3];
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
            console.warn("[Vango] Unknown frame type:", frameType);
          }
      }
    }
    /**
     * Handle patches frame
     */
    _handlePatches(buffer) {
      if (this.options.debug) {
        console.log("[Vango] Received patches buffer, length:", buffer.length);
      }
      const { seq, patches } = this.codec.decodePatches(buffer);
      if (this.options.debug) {
        console.log("[Vango] Decoded", patches.length, "patches, seq:", seq);
      }
      if (seq < this.expectedPatchSeq) {
        if (this.options.debug) {
          console.log("[Vango] Ignoring duplicate patch seq:", seq, "expected:", this.expectedPatchSeq);
        }
        return;
      }
      if (seq > this.expectedPatchSeq) {
        console.warn(
          "[Vango] Patch sequence gap detected",
          "expected:",
          this.expectedPatchSeq,
          "received:",
          seq
        );
        this._requestResync(this.patchSeq);
        return;
      }
      if (this.options.debug) {
        for (const p of patches) {
          console.log("[Vango] Patch:", p);
        }
        console.log("[Vango] Applying", patches.length, "patches (seq:", seq, ")");
      }
      this.optimistic.clearPending();
      this.patchApplier.apply(patches);
      this.hooks.updateFromDOM();
      this.patchSeq = seq;
      this.expectedPatchSeq = seq + 1;
      this.pendingResync = false;
      this._sendAck(seq);
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
        case ControlType.RESYNC_FULL:
          this._handleResyncFull(buffer.slice(1));
          break;
        case ControlType.CLOSE:
          this.wsManager.close();
          break;
        default:
          if (this.options.debug) {
            console.log("[Vango] Unknown control type:", controlType);
          }
      }
    }
    /**
     * Handle ResyncFull - replace body content with server-sent HTML
     * Used during session resume to ensure client DOM matches server state
     */
    _handleResyncFull(buffer) {
      if (buffer.length === 0)
        return;
      const { value: html } = this.codec.decodeString(buffer, 0);
      if (this.options.debug) {
        console.log("[Vango] ResyncFull received, replacing body content");
      }
      const template = document.createElement("template");
      template.innerHTML = html;
      this.nodeMap.clear();
      this.hooks.destroyAll();
      document.body.innerHTML = "";
      while (template.content.firstChild) {
        document.body.appendChild(template.content.firstChild);
      }
      this._buildNodeMap();
      this.hooks.initializeFromDOM();
      this.patchSeq = 0;
      this.expectedPatchSeq = 1;
      this.pendingResync = false;
    }
    /**
     * Send ACK for received patches
     * Format: [lastSeq:varint][window:varint]
     */
    _sendAck(lastSeq) {
      if (!this.connected)
        return;
      const payload = this.codec.encodeAck(lastSeq, 100);
      const frame = this._encodeFrame(FrameType.ACK, payload);
      this.wsManager.send(frame);
      if (this.options.debug) {
        console.log("[Vango] Sent ACK for seq:", lastSeq);
      }
    }
    /**
     * Request resync for missed patches (with debouncing)
     * Format: [controlType:1][lastSeq:varint]
     */
    _requestResync(lastSeq) {
      if (this.pendingResync) {
        if (this.options.debug) {
          console.log("[Vango] Resync already pending, skipping");
        }
        return;
      }
      this.pendingResync = true;
      if (this.options.debug) {
        console.log("[Vango] Requesting resync from seq:", lastSeq);
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
      if (buffer.length < 3)
        return;
      const code = buffer[0] << 8 | buffer[1];
      const { value: message, bytesRead } = this.codec.decodeString(buffer, 2);
      const fatalOffset = 2 + bytesRead;
      const fatal = fatalOffset < buffer.length ? buffer[fatalOffset] === 1 : false;
      const errorMessages = {
        0: "Unknown error",
        1: "Invalid frame",
        2: "Invalid event",
        3: "Handler not found",
        4: "Handler panic",
        5: "Session expired",
        6: "Rate limited",
        256: "Server error",
        257: "Not authorized",
        258: "Not found",
        259: "Validation failed",
        260: "Route error"
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
      const frame = this._encodeFrame(FrameType.EVENT, eventBuffer);
      this.wsManager.send(frame);
      if (this.options.debug) {
        console.log("[Vango] Sent event:", { type, hid, data, seq: this.seq });
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
      frame[1] = 0;
      frame[2] = length >> 8 & 255;
      frame[3] = length & 255;
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
