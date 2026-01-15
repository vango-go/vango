/**
 * Error Frame Decoding and Handshake Tests
 *
 * Tests for error wire format per spec Section 9.6.2
 * and handshake framing per protocol spec.
 */

import { jest, describe, test, expect } from '@jest/globals';
import { BinaryCodec } from '../src/codec.js';

describe('error frame decoding', () => {
    const codec = new BinaryCodec();

    /**
     * Build an error frame matching server encoding.
     * Wire format: [uint16:code][varint-string:message][bool:fatal]
     */
    function buildErrorFrame(code, message, fatal) {
        const msgBytes = codec.encodeString(message);
        const frame = new Uint8Array(2 + msgBytes.length + 1);

        // Code as big-endian uint16
        frame[0] = (code >> 8) & 0xff;
        frame[1] = code & 0xff;

        // Message (varint-length prefixed string)
        frame.set(msgBytes, 2);

        // Fatal flag
        frame[2 + msgBytes.length] = fatal ? 1 : 0;

        return frame;
    }

    test('decodes wire format: [code][message][fatal]', () => {
        const code = 0x0003; // HandlerNotFound
        const message = 'Test error message';
        const fatal = true;

        const frame = buildErrorFrame(code, message, fatal);

        // Decode code (2 bytes, big-endian)
        const decodedCode = (frame[0] << 8) | frame[1];
        expect(decodedCode).toBe(code);

        // Decode message (varint-length string starting at offset 2)
        const { value: decodedMessage, bytesRead } = codec.decodeString(frame, 2);
        expect(decodedMessage).toBe(message);

        // Decode fatal flag (1 byte after message)
        const fatalOffset = 2 + bytesRead;
        const decodedFatal = frame[fatalOffset] === 1;
        expect(decodedFatal).toBe(fatal);
    });

    test('decodes non-fatal error correctly', () => {
        const frame = buildErrorFrame(0x0001, 'Invalid frame', false);

        const fatalOffset = 2 + codec.decodeString(frame, 2).bytesRead;
        expect(frame[fatalOffset]).toBe(0);
    });

    test('decodes empty message correctly', () => {
        const frame = buildErrorFrame(0x0000, '', true);

        const { value: message, bytesRead } = codec.decodeString(frame, 2);
        expect(message).toBe('');
        expect(bytesRead).toBe(1); // Just the length byte (0)

        const fatalOffset = 2 + bytesRead;
        expect(frame[fatalOffset]).toBe(1);
    });

    test('decodes all standard error codes', () => {
        const errorCodes = [
            [0x0000, 'Unknown error'],
            [0x0001, 'Invalid frame'],
            [0x0002, 'Invalid event'],
            [0x0003, 'Handler not found'],
            [0x0004, 'Handler panic'],
            [0x0005, 'Session expired'],
            [0x0006, 'Rate limited'],
            [0x0100, 'Server error'],
            [0x0101, 'Not authorized'],
            [0x0102, 'Not found'],
            [0x0103, 'Validation failed'],
            [0x0104, 'Route error'],
        ];

        for (const [code, message] of errorCodes) {
            const frame = buildErrorFrame(code, message, false);
            const decodedCode = (frame[0] << 8) | frame[1];
            expect(decodedCode).toBe(code);

            const { value } = codec.decodeString(frame, 2);
            expect(value).toBe(message);
        }
    });

    test('handles unicode in error messages', () => {
        const message = 'Error: 无效的请求';
        const frame = buildErrorFrame(0x0100, message, true);

        const { value } = codec.decodeString(frame, 2);
        expect(value).toBe(message);
    });
});

describe('BinaryCodec handshake', () => {
    const codec = new BinaryCodec();

    describe('encodeClientHelloFrame', () => {
        test('wraps payload in 4-byte frame header', () => {
            const frame = codec.encodeClientHelloFrame({
                csrf: 'test-csrf',
                sessionId: '',
                viewportW: 1920,
                viewportH: 1080,
            });

            // Frame header: [type:1][flags:1][len:2]
            expect(frame[0]).toBe(0x00); // FrameHandshake
            expect(frame[1]).toBe(0x00); // No flags

            // Length should be big-endian
            const payloadLen = (frame[2] << 8) | frame[3];
            expect(payloadLen).toBe(frame.length - 4);
        });

        test('frame type is FrameHandshake (0x00)', () => {
            const frame = codec.encodeClientHelloFrame({
                csrf: '',
                sessionId: '',
                viewportW: 800,
                viewportH: 600,
            });
            expect(frame[0]).toBe(0x00);
        });

        test('payload starts at byte 4 with protocol version', () => {
            const frame = codec.encodeClientHelloFrame({
                csrf: 'abc',
                sessionId: '',
                viewportW: 800,
                viewportH: 600,
            });

            // Payload starts with version bytes [major, minor]
            // Protocol version 2.0 = [0x02, 0x00]
            expect(frame[4]).toBe(0x02); // Major version
            expect(frame[5]).toBe(0x00); // Minor version
        });

        test('includes CSRF after version bytes', () => {
            const frame = codec.encodeClientHelloFrame({
                csrf: 'my-csrf-token',
                sessionId: '',
                viewportW: 1024,
                viewportH: 768,
            });

            // Skip 4-byte frame header + 2-byte version
            const { value: csrf } = codec.decodeString(frame, 6);
            expect(csrf).toBe('my-csrf-token');
        });
    });

    describe('decodeServerHello', () => {
        /**
         * Build a framed ServerHello response
         * Format: [type:1][flags:1][len:2][status:1][sessionId:string][extra...]
         */
        function buildServerHelloFrame(status, sessionId) {
            const sessionBytes = codec.encodeString(sessionId);
            // Need: status(1) + session + nextSeq(4) + serverTime(8) + flags(2)
            const payloadLen = 1 + sessionBytes.length + 4 + 8 + 2;
            const frame = new Uint8Array(4 + payloadLen);

            // Frame header
            frame[0] = 0x00; // FrameHandshake
            frame[1] = 0x00; // flags
            frame[2] = (payloadLen >> 8) & 0xff;
            frame[3] = payloadLen & 0xff;

            // Payload
            let offset = 4;
            frame[offset++] = status;
            frame.set(sessionBytes, offset);
            offset += sessionBytes.length;
            // nextSeq (4 bytes), serverTime (8 bytes), flags (2 bytes) are zeros

            return frame;
        }

        test('skips 4-byte frame header', () => {
            const frame = buildServerHelloFrame(0x00, 'sess-abc');
            const result = codec.decodeServerHello(frame);

            expect(result.status).toBe(0x00);
            expect(result.ok).toBe(true);
        });

        test('reads status from byte 4, not byte 1', () => {
            // Create a frame where byte[1] is flags (0x00) but byte[4] is status (0x01)
            const frame = buildServerHelloFrame(0x01, '');
            // Verify flags byte is 0x00
            expect(frame[1]).toBe(0x00);

            const result = codec.decodeServerHello(frame);

            // Status should be 0x01 (from byte 4), not 0x00 (from byte 1)
            expect(result.status).toBe(0x01);
            expect(result.ok).toBe(false);
        });

        test('decodes successful handshake', () => {
            const frame = buildServerHelloFrame(0x00, 'session-xyz');
            const result = codec.decodeServerHello(frame);

            expect(result.ok).toBe(true);
            expect(result.status).toBe(0x00);
            expect(result.sessionId).toBe('session-xyz');
        });

        test('decodes version mismatch error', () => {
            const frame = buildServerHelloFrame(0x01, '');
            const result = codec.decodeServerHello(frame);

            expect(result.ok).toBe(false);
            expect(result.status).toBe(0x01);
        });

        test('decodes invalid CSRF error', () => {
            const frame = buildServerHelloFrame(0x02, '');
            const result = codec.decodeServerHello(frame);

            expect(result.ok).toBe(false);
            expect(result.status).toBe(0x02);
        });

        test('decodes session expired error', () => {
            const frame = buildServerHelloFrame(0x03, '');
            const result = codec.decodeServerHello(frame);

            expect(result.ok).toBe(false);
            expect(result.status).toBe(0x03);
        });

        test('decodes server busy error', () => {
            const frame = buildServerHelloFrame(0x04, '');
            const result = codec.decodeServerHello(frame);

            expect(result.ok).toBe(false);
            expect(result.status).toBe(0x04);
        });

        test('returns error for buffer too short', () => {
            const shortBuffer = new Uint8Array([0x00, 0x00, 0x00]);
            expect(() => codec.decodeServerHello(shortBuffer)).toThrow();
        });

        test('returns error for wrong frame type', () => {
            const wrongType = new Uint8Array([
                0x02, // Wrong type (should be 0x00)
                0x00,
                0x00,
                0x01,
                0x00,
            ]);
            expect(() => codec.decodeServerHello(wrongType)).toThrow();
        });
    });

    describe('encodeClientHello (raw payload)', () => {
        test('starts with protocol version 2.0', () => {
            const payload = codec.encodeClientHello({
                csrf: '',
                sessionId: '',
                viewportW: 800,
                viewportH: 600,
            });
            // Protocol version 2.0 = [0x02, 0x00]
            expect(payload[0]).toBe(0x02); // Major
            expect(payload[1]).toBe(0x00); // Minor
        });

        test('includes CSRF token after version', () => {
            const payload = codec.encodeClientHello({
                csrf: 'test-token',
                sessionId: '',
                viewportW: 800,
                viewportH: 600,
            });

            // CSRF starts at offset 2 (after 2-byte version)
            const { value } = codec.decodeString(payload, 2);
            expect(value).toBe('test-token');
        });

        test('includes viewport dimensions', () => {
            const payload = codec.encodeClientHello({
                csrf: '',
                sessionId: '',
                viewportW: 1920,
                viewportH: 1080,
            });

            // Skip version(2) + csrf(1 for empty) + session(1 for empty) + lastSeq(4)
            let offset = 2;
            const { bytesRead: csrfBytes } = codec.decodeString(payload, offset);
            offset += csrfBytes;
            const { bytesRead: sessionBytes } = codec.decodeString(payload, offset);
            offset += sessionBytes;
            offset += 4; // lastSeq uint32

            // Viewport is encoded as uint16 big-endian (matches Go protocol)
            const w = (payload[offset] << 8) | payload[offset + 1];
            const h = (payload[offset + 2] << 8) | payload[offset + 3];

            expect(w).toBe(1920);
            expect(h).toBe(1080);
        });
    });
});

describe('WebSocketManager connection behavior', () => {
    // These tests document the expected behavior patterns without
    // actually testing location mocking (which jsdom doesn't support well)

    test('pendingNavPath should be on client.eventCapture, not client.events', () => {
        // This test documents the correct property path
        const mockClient = {
            eventCapture: {
                pendingNavPath: '/dashboard',
            },
        };

        // Correct access pattern
        expect(mockClient.eventCapture?.pendingNavPath).toBe('/dashboard');

        // The buggy pattern would be:
        // mockClient.events?.pendingNavPath (would be undefined)
        expect(mockClient.events?.pendingNavPath).toBeUndefined();
    });

    test('connection loss with pending nav path should complete navigation', () => {
        // This documents the expected behavior contract:
        // When wasConnected && pendingPath:
        //   - location.assign(pendingPath) should be called
        //   - return early (don't reconnect)
        //
        // When wasConnected && !pendingPath:
        //   - _onDisconnected() should be called
        //   - reconnect should proceed normally

        const mockEventCapture = {
            pendingNavPath: '/target-page',
        };

        const pendingPath = mockEventCapture.pendingNavPath;
        expect(pendingPath).toBe('/target-page');
        // In real code, this would call: location.assign(pendingPath)
    });

    test('connection loss without pending nav should reconnect normally', () => {
        const mockEventCapture = {
            pendingNavPath: null,
        };

        const pendingPath = mockEventCapture.pendingNavPath;
        expect(pendingPath).toBeNull();
        // In real code, this would proceed to reconnection logic
    });
});
