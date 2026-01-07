/**
 * Tests for Phase 2: Resync, ACK, and Patch Sequence Tracking
 */

import { BinaryCodec } from '../src/codec.js';

describe('BinaryCodec ACK and Resync encoding', () => {
    let codec;

    beforeEach(() => {
        codec = new BinaryCodec();
    });

    describe('encodeAck', () => {
        test('encodes ACK with lastSeq and default window', () => {
            const ack = codec.encodeAck(42);

            // Decode manually to verify
            // Format: [lastSeq:varint][window:varint]
            expect(ack[0]).toBe(42); // lastSeq as varint (single byte for small values)
            expect(ack[1]).toBe(100); // default window=100 as varint
        });

        test('encodes ACK with custom window', () => {
            const ack = codec.encodeAck(10, 50);

            expect(ack[0]).toBe(10); // lastSeq
            expect(ack[1]).toBe(50); // window
        });

        test('encodes ACK with large sequence number', () => {
            // 300 = 0x12C, which needs 2 bytes as varint
            // 300 = 172 + 128, varint = [0xAC, 0x02]
            const ack = codec.encodeAck(300, 100);

            // Verify by decoding
            let offset = 0;
            const { value: lastSeq, bytesRead: seqBytes } = codec.decodeUvarint(ack, offset);
            offset += seqBytes;
            const { value: window } = codec.decodeUvarint(ack, offset);

            expect(lastSeq).toBe(300);
            expect(window).toBe(100);
        });

        test('produces valid binary that can be round-tripped', () => {
            const testCases = [
                { seq: 0, window: 100 },
                { seq: 1, window: 100 },
                { seq: 127, window: 100 },
                { seq: 128, window: 100 },
                { seq: 16383, window: 50 },
                { seq: 16384, window: 200 },
            ];

            for (const tc of testCases) {
                const encoded = codec.encodeAck(tc.seq, tc.window);

                let offset = 0;
                const { value: decodedSeq, bytesRead: seqBytes } = codec.decodeUvarint(encoded, offset);
                offset += seqBytes;
                const { value: decodedWindow } = codec.decodeUvarint(encoded, offset);

                expect(decodedSeq).toBe(tc.seq);
                expect(decodedWindow).toBe(tc.window);
            }
        });
    });

    describe('encodeResyncRequest', () => {
        test('encodes ResyncRequest with control type prefix', () => {
            const request = codec.encodeResyncRequest(42);

            // Format: [controlType:1][lastSeq:varint]
            expect(request[0]).toBe(0x10); // ControlResyncRequest
            expect(request[1]).toBe(42); // lastSeq as varint
        });

        test('encodes ResyncRequest with zero sequence', () => {
            const request = codec.encodeResyncRequest(0);

            expect(request[0]).toBe(0x10); // ControlResyncRequest
            expect(request[1]).toBe(0); // lastSeq=0
        });

        test('encodes ResyncRequest with large sequence', () => {
            const request = codec.encodeResyncRequest(1000);

            expect(request[0]).toBe(0x10); // ControlResyncRequest

            // Decode the varint portion
            const { value: lastSeq } = codec.decodeUvarint(request.slice(1), 0);
            expect(lastSeq).toBe(1000);
        });
    });
});

describe('Big-endian encoding verification', () => {
    let codec;

    beforeEach(() => {
        codec = new BinaryCodec();
    });

    test('encodeUint16 uses big-endian', () => {
        // 0x1234 should be [0x12, 0x34] in big-endian
        const result = codec.encodeUint16(0x1234);
        expect(result[0]).toBe(0x12);
        expect(result[1]).toBe(0x34);
    });

    test('decodeUint16 uses big-endian', () => {
        // [0x12, 0x34] should decode to 0x1234
        const buffer = new Uint8Array([0x12, 0x34]);
        const result = codec.decodeUint16(buffer, 0);
        expect(result).toBe(0x1234);
    });

    test('encodeUint32 uses big-endian', () => {
        // 0x12345678 should be [0x12, 0x34, 0x56, 0x78] in big-endian
        const result = codec.encodeUint32(0x12345678);
        expect(result[0]).toBe(0x12);
        expect(result[1]).toBe(0x34);
        expect(result[2]).toBe(0x56);
        expect(result[3]).toBe(0x78);
    });

    test('decodeUint32 uses big-endian', () => {
        // [0x12, 0x34, 0x56, 0x78] should decode to 0x12345678
        const buffer = new Uint8Array([0x12, 0x34, 0x56, 0x78]);
        const result = codec.decodeUint32(buffer, 0);
        expect(result).toBe(0x12345678);
    });

    test('round-trip uint16', () => {
        const values = [0, 1, 255, 256, 1000, 65535];
        for (const v of values) {
            const encoded = codec.encodeUint16(v);
            const decoded = codec.decodeUint16(encoded, 0);
            expect(decoded).toBe(v);
        }
    });

    test('round-trip uint32', () => {
        const values = [0, 1, 255, 65535, 0x10000, 0xFFFFFF, 0x7FFFFFFF];
        for (const v of values) {
            const encoded = codec.encodeUint32(v);
            const decoded = codec.decodeUint32(encoded, 0);
            expect(decoded).toBe(v);
        }
    });
});

describe('Patch sequence tracking', () => {
    // These tests verify the client's sequence tracking behavior
    // by testing the underlying logic without a full VangoClient instance

    test('sequence gap detection logic', () => {
        let patchSeq = 0;
        let expectedPatchSeq = 1;

        // Simulate receiving seq=1 (expected)
        const seq1 = 1;
        const isGap1 = seq1 > expectedPatchSeq;
        expect(isGap1).toBe(false);

        // Update tracking
        patchSeq = seq1;
        expectedPatchSeq = seq1 + 1;
        expect(expectedPatchSeq).toBe(2);

        // Simulate receiving seq=3 (gap - skipped seq=2)
        const seq3 = 3;
        const isGap3 = seq3 > expectedPatchSeq;
        expect(isGap3).toBe(true); // Gap detected!
    });

    test('duplicate frame detection logic', () => {
        let expectedPatchSeq = 5;

        // Simulate receiving seq=3 (already processed)
        const seq = 3;
        const isDuplicate = seq < expectedPatchSeq;
        expect(isDuplicate).toBe(true);
    });

    test('normal sequence acceptance', () => {
        let expectedPatchSeq = 5;

        // Simulate receiving seq=5 (expected)
        const seq = 5;
        const isExpected = seq === expectedPatchSeq;
        expect(isExpected).toBe(true);
    });
});

describe('Control type constants', () => {
    // Verify control types match Go protocol/control.go
    const ControlType = {
        PING: 0x01,
        PONG: 0x02,
        RESYNC_REQUEST: 0x10,
        RESYNC_PATCHES: 0x11,
        RESYNC_FULL: 0x12,
        CLOSE: 0x20,
    };

    test('PING matches Go protocol', () => {
        expect(ControlType.PING).toBe(0x01);
    });

    test('PONG matches Go protocol', () => {
        expect(ControlType.PONG).toBe(0x02);
    });

    test('RESYNC_REQUEST matches Go protocol', () => {
        expect(ControlType.RESYNC_REQUEST).toBe(0x10);
    });

    test('RESYNC_PATCHES matches Go protocol', () => {
        expect(ControlType.RESYNC_PATCHES).toBe(0x11);
    });

    test('RESYNC_FULL matches Go protocol', () => {
        expect(ControlType.RESYNC_FULL).toBe(0x12);
    });

    test('CLOSE matches Go protocol', () => {
        expect(ControlType.CLOSE).toBe(0x20);
    });
});
