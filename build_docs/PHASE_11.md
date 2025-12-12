# Phase 11: Security Hardening

> **Hardening the framework against identified vulnerabilities**

---

## Overview

Phase 11 addresses security vulnerabilities identified during a comprehensive security audit. The focus is on defense-in-depth: preventing DoS attacks, eliminating XSS/RCE vectors, and enforcing secure defaults that protect developers from common mistakes.

### Goals

1. **Protocol Hardening**: Prevent DoS via malicious length prefixes
2. **Secure Defaults**: Same-origin WebSocket, proper CSRF
3. **XSS Elimination**: Remove arbitrary JS execution vectors
4. **Defense in Depth**: Multiple layers of protection

### Threat Model

- Attacker can send arbitrary WebSocket frames/events
- Application code may accidentally pass untrusted data to VNodes
- Developers may forget to configure security settings

---

## Subsystems

| Subsystem | Purpose | Priority |
|-----------|---------|----------|
| 11.1 Protocol Decoder Hardening | Prevent DoS via allocation abuse | Critical |
| 11.2 Secure Server Defaults | Same-origin, CSRF enforcement | Critical |
| 11.3 PatchEval Removal | Eliminate client-side RCE | Critical |
| 11.4 Attribute Sanitization | Prevent XSS via on* injection | High |
| 11.5 CSRF Double Submit Cookie | Production-ready CSRF protection | High |

---

## 11.1 Protocol Decoder Hardening

### Problem

Varint lengths/counts were trusted without bounds checking. A malicious frame could encode huge lengths that cause:
- OOM panics via `make([]byte, hugeLength)`
- Integer overflow on `int(length)` conversion
- Server crash (no recovery in WS read path)

### Solution

Added allocation limits and bounds checking to `pkg/protocol/decoder.go`.

### Constants

```go
const (
    DefaultMaxAllocation = 4 * 1024 * 1024   // 4MB default
    HardMaxAllocation    = 16 * 1024 * 1024  // 16MB absolute cap
    MaxCollectionCount   = 100_000           // Max array/map items
)
```

### New Errors

```go
var (
    ErrAllocationTooLarge = errors.New("protocol: allocation size exceeds limit")
    ErrCollectionTooLarge = errors.New("protocol: collection count exceeds limit")
)
```

### Changes

| File | Change |
|------|--------|
| `pkg/protocol/decoder.go` | Added `ReadCollectionCount()`, bounds checks in `ReadString()`/`ReadLenBytes()` |
| `pkg/protocol/event.go` | Replaced `ReadUvarint()` with `ReadCollectionCount()` for all collections |

### Exit Criteria

- [x] `ReadString()` checks `length <= Remaining()` and `length <= DefaultMaxAllocation`
- [x] `ReadLenBytes()` same bounds checking
- [x] `ReadCollectionCount()` validates count against `MaxCollectionCount`
- [x] All collection reads in `event.go` use safe counting

---

## 11.2 Secure Server Defaults

### Problem

Default configuration was insecure:
- `CheckOrigin` allowed all origins (Cross-Site WebSocket Hijacking)
- `CSRFSecret` was nil (CSRF disabled)
- CSRF validation used `RemoteAddr` (unstable behind proxies)

### Solution

Changed defaults in `pkg/server/config.go` and added CSRF warning.

### Changes

| Setting | Before | After |
|---------|--------|-------|
| `CheckOrigin` | `return true` (allow all) | `SameOriginCheck` (reject cross-origin) |
| `CSRFSecret` | `nil` (disabled) | `nil` (warning logged on startup) |

### New Functions

```go
// SameOriginCheck validates that the WebSocket request origin matches the host.
func SameOriginCheck(r *http.Request) bool

// SetCSRFCookie sets the __vango_csrf cookie for Double Submit pattern.
func (s *Server) SetCSRFCookie(w http.ResponseWriter, token string)
```

### Startup Warning

```
WARN CSRF protection is DISABLED. Set CSRFSecret for production use. This will become a hard requirement in Vango v3.0.
```

---

## 11.3 PatchEval Removal

### Problem

`PatchEval` (0x21) allowed the server to send arbitrary JavaScript to the client for execution via `new Function()`. This is a catastrophic XSS/RCE vulnerability.

### Solution

**Completely removed** `PatchEval` from both Go and JavaScript.

### Changes

| File | Change |
|------|--------|
| `pkg/protocol/patch.go` | Removed `PatchEval` constant, encoder case, decoder case, `NewEvalPatch()` |
| `client/src/patches.js` | Removed `PatchType.EVAL` handler, `_evalCode()` method |
| `pkg/protocol/patch_test.go` | Removed eval test cases |

### Migration

```go
// BEFORE (removed)
patch := NewEvalPatch("h1", "console.log('hello')")

// AFTER (use dispatch + client hook)
patch := NewDispatchPatch("h1", "vango:log", `{"message":"hello"}`)
```

---

## 11.4 Attribute Sanitization

### Problem

The renderer could output `on*` attributes as strings if developers accidentally set them. An attacker could inject `onclick="alert(1)"` via user input.

### Solution

Case-insensitive filtering of ALL `on*` attributes that are not valid internal EventHandlers.

### Implementation

```go
// isEventHandlerKey returns true for onclick, ONCLICK, onClick, etc.
func isEventHandlerKey(key string) bool {
    return len(key) > 2 && strings.EqualFold(key[:2], "on")
}
```

### Behavior

- If key matches `on*` AND value is a valid Go function → registered as handler (not rendered)
- If key matches `on*` AND value is NOT a function → **silently stripped**

---

## 11.5 CSRF Double Submit Cookie

### Problem

Previous CSRF implementation used `RemoteAddr` for token generation, which is:
- Unstable behind proxies/NATs
- Not secret (can be discovered)
- Not tied to session

### Solution

Implemented Double Submit Cookie pattern.

### Flow

```
1. Server: GenerateCSRFToken() → random 32-byte token
2. Server: SetCSRFCookie(w, token) → sets __vango_csrf cookie
3. Server: Embeds token in HTML as window.__VANGO_CSRF__ or cookie
4. Client: Reads token from window.__VANGO_CSRF__ or cookie
5. Client: Sends token in WebSocket handshake
6. Server: validateCSRF() compares handshake token with cookie value
```

### Client Changes

```javascript
// client/src/websocket.js
_getCSRFToken() {
    if (window.__VANGO_CSRF__) {
        return window.__VANGO_CSRF__;
    }
    const match = document.cookie.match(/(?:^|;\s*)__vango_csrf=([^;]*)/);
    return match ? decodeURIComponent(match[1]) : '';
}
```

---

## Breaking Changes

| Change | Impact | Migration |
|--------|--------|-----------|
| `NewEvalPatch()` removed | Compile error | Use `PatchDispatch` + client hooks |
| `CheckOrigin` default | Cross-origin WS rejected | Configure `CheckOrigin` explicitly |
| `on*` attributes stripped | Inline handlers removed | Use `OnClick()` helpers |

---

## Testing

| Test | Result |
|------|--------|
| `pkg/protocol/...` | ✅ PASS |
| `pkg/server/...` | ✅ PASS |
| `go build ./...` | ✅ PASS |

---

## Files Changed

### Protocol Layer
- `pkg/protocol/decoder.go` - Allocation limits, bounds checking
- `pkg/protocol/event.go` - Safe collection reading
- `pkg/protocol/patch.go` - PatchEval removal
- `pkg/protocol/patch_test.go` - Test updates

### Server Layer
- `pkg/server/config.go` - Secure defaults, SameOriginCheck
- `pkg/server/server.go` - CSRF warning, Double Submit Cookie

### Client Layer
- `client/src/patches.js` - Removed eval handler
- `client/src/websocket.js` - CSRF cookie reading

### Render Layer
- `pkg/render/renderer.go` - Attribute sanitization

---

## Future Work

- Fuzz testing for decoder bounds
- Integration tests for CSRF flow
- Per-session random HID prefix (reduces HID guessability)
- Render panic recovery

---

## Follow-Up Fixes (Audit Round 2)

After an additional security review, the following gaps were addressed:

### Critical: ControlResyncPatches DoS
- `pkg/protocol/control.go` - Changed `ReadUvarint()` to `ReadCollectionCount()` for patch count

### High: Complete Case-Insensitive on* Filtering
- `pkg/vdom/diff.go` - Updated `isEventHandler()` to use `strings.EqualFold`
- `pkg/protocol/vnode.go` - Updated wire encoding to use case-insensitive check
- `client/src/patches.js` - Added defensive `_setAttr()` on* blocking

### Medium: Path Traversal Prevention
- `pkg/upload/disk.go` - Added `isValidTempID()` validation (hex-only)
- `pkg/upload/disk.go` - Added path prefix check after `filepath.Abs()`

### Low: URL Parsing and Error Handling
- `pkg/server/config.go` - Replaced string-based origin parsing with `net/url.Parse()`
- `pkg/server/server.go` - Added `rand.Read()` error handling (panic on failure)
- `pkg/server/session.go` - Added `rand.Read()` error handling (panic on failure)
- `pkg/upload/disk.go` - Added `rand.Read()` error handling (panic on failure)

### Medium: CSRFSecret HMAC Signing
- `pkg/server/server.go` - `GenerateCSRFToken()` now HMAC-signs tokens with `CSRFSecret`
- `pkg/server/server.go` - `validateCSRF()` now verifies HMAC signatures
- Token format: `base64(16-byte nonce + 32-byte HMAC-SHA256 signature)`
- Backward compatible: if `CSRFSecret` is nil, falls back to unsigned tokens

---

## Follow-Up Fixes (Audit Round 3)

Additional hardening from third-party security review:

### Critical: Hook Payload Depth Limit
- `pkg/protocol/event.go` - Added `MaxHookDepth = 64` constant
- `decodeHookValue()` now tracks nesting depth and returns `ErrMaxDepthExceeded`
- Prevents stack overflow from deeply nested JSON in hook events

### Critical: Upload DoS Prevention
- `pkg/upload/upload.go` - Added `http.MaxBytesReader()` wrapper
- Request body is limited BEFORE `ParseMultipartForm()` parses
- New `HandlerWithConfig()` for custom max file size

### Critical: Remaining Collection Count Fixes
- `pkg/protocol/patch.go` - `DecodePatchesFrom()` now uses `ReadCollectionCount()`
- `pkg/protocol/vnode.go` - All `attrCount`/`childCount` now use `ReadCollectionCount()`

### High: Session Limits Wiring
- `pkg/server/server.go` - Config `MaxSessions` and `MaxMemoryPerSession` now applied
- Previously, these config values were ignored and defaults were always used

### Medium: Client Defense-in-Depth
- `client/src/optimistic.js` - Added on* blocking to `_applyAttrOptimistic()`
- `client/src/codec.js` - Removed `PatchType.EVAL` constant and decode case entirely

---

## Follow-Up Fixes (Audit Round 4)

Final cleanup and documentation improvements:

### Medium: Debug Log Gating
- `pkg/server/session.go` - All `fmt.Printf` gated behind `DebugMode`
- `pkg/server/websocket.go` - Event frame logging gated behind `DebugMode`
- `pkg/server/component.go` - MarkDirty logging gated behind `DebugMode`
- `pkg/render/renderer.go` - Removed unconditional HID debug prints
- `pkg/vdom/hydration.go` - Removed unconditional HID debug prints
- `client/src/index.js` - Patch logging gated behind `options.debug`

### Medium: AllowedTypes Enforcement
- `pkg/upload/upload.go` - Added `isTypeAllowed()` check in handler
- `pkg/upload/upload.go` - Added `ErrTypeNotAllowed` error
- MIME type matching is case-insensitive
- Returns `415 Unsupported Media Type` for blocked types

### Low: Raw HTML Documentation
- `pkg/vdom/helpers.go` - Enhanced `Raw()` doc comment with security warning
- Clear guidance: never pass user-provided strings directly

### Low: Test Updates
- Updated render tests to expect `data-hid` on all elements (correct VDOM behavior)
- Updated hydration tests for universal HID assignment
- Fixed `isEventHandler` test: `"on"` alone is not a valid event handler

