# Protocol Reference

The binary protocol between client and server.

## WebSocket Handshake

```json
// Client → Server
{
    "type": "HANDSHAKE",
    "version": "1.0",
    "csrf": "<token>",
    "session": "<session-id>"
}

// Server → Client
{
    "type": "HANDSHAKE_ACK",
    "session": "<session-id>"
}
```

## Event Format (Client → Server)

```
[Type: 1 byte] [HID: varint] [Payload: variable]
```

| Type | Code | Payload |
|------|------|---------|
| CLICK | 0x01 | none |
| INPUT | 0x03 | [length][utf8 string] |
| SUBMIT | 0x05 | [form encoding] |
| KEYDOWN | 0x08 | [keycode][modifiers] |
| NAVIGATE | 0x0D | [length][utf8 path] |

## Patch Format (Server → Client)

```
[Count: varint] [Patch 1] [Patch 2] ...

Each Patch:
[Type: 1 byte] [HID: varint] [Payload: variable]
```

| Type | Code | Payload |
|------|------|---------|
| SET_TEXT | 0x01 | [length][utf8 text] |
| SET_ATTR | 0x02 | [key][value] |
| REMOVE_ATTR | 0x03 | [key] |
| INSERT_NODE | 0x04 | [index][encoded vnode] |
| REMOVE_NODE | 0x05 | none |
| REPLACE_NODE | 0x0B | [encoded vnode] |

## Varint Encoding

Variable-length integers (Protocol Buffers style):
- 0-127: 1 byte
- 128-16383: 2 bytes
- 16384+: 3+ bytes

## VNode Encoding

```
[NodeType: 1 byte] ...

Element (0x01):
  [tag-length][tag][hid][attr-count][attrs...][child-count][children...]

Text (0x02):
  [length][utf8 text]

Fragment (0x03):
  [child-count][children...]
```
