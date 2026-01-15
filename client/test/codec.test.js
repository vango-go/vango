/**
 * Codec Unit Tests
 */

import { BinaryCodec, EventType, PatchType } from '../src/codec.js';

describe('BinaryCodec', () => {
    const codec = new BinaryCodec();

    describe('varint encoding', () => {
        test('encodes small numbers in one byte', () => {
            const encoded = codec.encodeUvarint(42);
            expect(encoded.length).toBe(1);
            expect(encoded[0]).toBe(42);
        });

        test('encodes zero correctly', () => {
            const encoded = codec.encodeUvarint(0);
            expect(encoded.length).toBe(1);
            expect(encoded[0]).toBe(0);
        });

        test('encodes 127 in one byte', () => {
            const encoded = codec.encodeUvarint(127);
            expect(encoded.length).toBe(1);
            expect(encoded[0]).toBe(127);
        });

        test('encodes 128 in two bytes', () => {
            const encoded = codec.encodeUvarint(128);
            expect(encoded.length).toBe(2);
            expect(encoded[0]).toBe(0x80); // 128 with continuation bit
            expect(encoded[1]).toBe(0x01);
        });

        test('encodes larger numbers in multiple bytes', () => {
            const encoded = codec.encodeUvarint(300);
            expect(encoded.length).toBe(2);
        });

        test('round-trips correctly', () => {
            const testValues = [0, 1, 127, 128, 255, 256, 16383, 16384, 100000, 1000000];
            for (const value of testValues) {
                const encoded = codec.encodeUvarint(value);
                const { value: decoded, bytesRead } = codec.decodeUvarint(encoded, 0);
                expect(decoded).toBe(value);
                expect(bytesRead).toBe(encoded.length);
            }
        });
    });

    describe('signed varint encoding', () => {
        test('encodes zero correctly', () => {
            const encoded = codec.encodeSvarint(0);
            const { value } = codec.decodeSvarint(encoded, 0);
            expect(value).toBe(0);
        });

        test('encodes positive numbers', () => {
            const encoded = codec.encodeSvarint(42);
            const { value } = codec.decodeSvarint(encoded, 0);
            expect(value).toBe(42);
        });

        test('encodes negative numbers', () => {
            const encoded = codec.encodeSvarint(-42);
            const { value } = codec.decodeSvarint(encoded, 0);
            expect(value).toBe(-42);
        });

        test('round-trips correctly', () => {
            const testValues = [0, 1, -1, 42, -42, 127, -128, 1000, -1000];
            for (const value of testValues) {
                const encoded = codec.encodeSvarint(value);
                const { value: decoded } = codec.decodeSvarint(encoded, 0);
                expect(decoded).toBe(value);
            }
        });
    });

    describe('string encoding', () => {
        test('encodes empty string', () => {
            const encoded = codec.encodeString('');
            const { value, bytesRead } = codec.decodeString(encoded, 0);
            expect(value).toBe('');
            expect(bytesRead).toBe(1); // Just length byte
        });

        test('encodes simple string', () => {
            const encoded = codec.encodeString('hello');
            const { value, bytesRead } = codec.decodeString(encoded, 0);
            expect(value).toBe('hello');
            expect(bytesRead).toBe(6); // 1 length byte + 5 chars
        });

        test('encodes unicode string', () => {
            const encoded = codec.encodeString('héllo 世界');
            const { value } = codec.decodeString(encoded, 0);
            expect(value).toBe('héllo 世界');
        });

        test('round-trips correctly', () => {
            const testStrings = ['', 'a', 'hello', 'Hello, World!', '日本語', '🎉'];
            for (const str of testStrings) {
                const encoded = codec.encodeString(str);
                const { value } = codec.decodeString(encoded, 0);
                expect(value).toBe(str);
            }
        });
    });

    describe('decode bounds checks', () => {
        test('rejects truncated varint', () => {
            const buffer = new Uint8Array([0x80]); // continuation bit without next byte
            expect(() => codec.decodeUvarint(buffer, 0)).toThrow();
        });

        test('rejects varint overflow', () => {
            const buffer = new Uint8Array(11);
            buffer.fill(0x80);
            expect(() => codec.decodeUvarint(buffer, 0)).toThrow();
        });

        test('rejects string length beyond remaining bytes', () => {
            const parts = [
                codec.encodeUvarint(5),
                new Uint8Array([0x61, 0x62]),
            ];
            const buffer = new Uint8Array(parts[0].length + parts[1].length);
            buffer.set(parts[0], 0);
            buffer.set(parts[1], parts[0].length);
            expect(() => codec.decodeString(buffer, 0)).toThrow();
        });

        test('rejects patch frames with missing payload', () => {
            const parts = [
                codec.encodeUvarint(1),
                codec.encodeUvarint(1),
            ];
            const buffer = new Uint8Array(parts[0].length + parts[1].length);
            buffer.set(parts[0], 0);
            buffer.set(parts[1], parts[0].length);
            expect(() => codec.decodePatches(buffer)).toThrow();
        });
    });

    describe('event encoding', () => {
        test('encodes click event', () => {
            const buffer = codec.encodeEvent(1, EventType.CLICK, 'h42', null);
            // Should have: seq(1 byte) + type(1 byte) + hid string(4 bytes for "h42")
            expect(buffer.length).toBe(6);
        });

        test('encodes input event with value', () => {
            const buffer = codec.encodeEvent(1, EventType.INPUT, 'h1', { value: 'hello' });
            expect(buffer.length).toBeGreaterThan(5);
        });

        test('encodes submit event with form data', () => {
            const buffer = codec.encodeEvent(1, EventType.SUBMIT, 'h1', {
                name: 'John',
                email: 'john@example.com'
            });
            expect(buffer.length).toBeGreaterThan(10);
        });

        test('encodes keyboard event', () => {
            const buffer = codec.encodeEvent(1, EventType.KEYDOWN, 'h1', {
                key: 'Enter',
                ctrlKey: true,
                shiftKey: false,
                altKey: false,
                metaKey: false,
            });
            expect(buffer.length).toBeGreaterThan(5);
        });
    });

    describe('patch decoding', () => {
        test('decodes SET_TEXT patch', () => {
            // Build a patches frame manually:
            // [seq:1][count:1][type:1][hid:"h5"][text:"hello"]
            const parts = [
                codec.encodeUvarint(1), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.SET_TEXT]),
                codec.encodeString('h5'),
                codec.encodeString('hello'),
            ];

            let totalLength = 0;
            for (const p of parts) totalLength += p.length;
            const buffer = new Uint8Array(totalLength);
            let offset = 0;
            for (const p of parts) {
                buffer.set(p, offset);
                offset += p.length;
            }

            const { seq, patches } = codec.decodePatches(buffer);

            expect(seq).toBe(1);
            expect(patches.length).toBe(1);
            expect(patches[0].type).toBe(PatchType.SET_TEXT);
            expect(patches[0].hid).toBe('h5');
            expect(patches[0].value).toBe('hello');
        });

        test('decodes SET_ATTR patch', () => {
            const parts = [
                codec.encodeUvarint(1), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.SET_ATTR]),
                codec.encodeString('h3'),
                codec.encodeString('class'),
                codec.encodeString('active'),
            ];

            let totalLength = 0;
            for (const p of parts) totalLength += p.length;
            const buffer = new Uint8Array(totalLength);
            let offset = 0;
            for (const p of parts) {
                buffer.set(p, offset);
                offset += p.length;
            }

            const { patches } = codec.decodePatches(buffer);

            expect(patches.length).toBe(1);
            expect(patches[0].type).toBe(PatchType.SET_ATTR);
            expect(patches[0].hid).toBe('h3');
            expect(patches[0].key).toBe('class');
            expect(patches[0].value).toBe('active');
        });

        test('decodes multiple patches', () => {
            const parts = [
                codec.encodeUvarint(5), // seq
                codec.encodeUvarint(2), // count = 2
                // First patch
                new Uint8Array([PatchType.SET_TEXT]),
                codec.encodeString('h1'),
                codec.encodeString('Hello'),
                // Second patch
                new Uint8Array([PatchType.ADD_CLASS]),
                codec.encodeString('h2'),
                codec.encodeString('visible'),
            ];

            let totalLength = 0;
            for (const p of parts) totalLength += p.length;
            const buffer = new Uint8Array(totalLength);
            let offset = 0;
            for (const p of parts) {
                buffer.set(p, offset);
                offset += p.length;
            }

            const { seq, patches } = codec.decodePatches(buffer);

            expect(seq).toBe(5);
            expect(patches.length).toBe(2);
            expect(patches[0].type).toBe(PatchType.SET_TEXT);
            expect(patches[0].value).toBe('Hello');
            expect(patches[1].type).toBe(PatchType.ADD_CLASS);
            expect(patches[1].className).toBe('visible');
        });
    });

    describe('VNode decoding', () => {
        test('decodes text node', () => {
            // VNode format: [type:0x02][text]
            const parts = [
                new Uint8Array([0x02]), // TEXT type
                codec.encodeString('Hello, World!'),
            ];

            let totalLength = 0;
            for (const p of parts) totalLength += p.length;
            const buffer = new Uint8Array(totalLength);
            let offset = 0;
            for (const p of parts) {
                buffer.set(p, offset);
                offset += p.length;
            }

            const { vnode, bytesRead } = codec.decodeVNode(buffer, 0);

            expect(vnode.type).toBe('text');
            expect(vnode.text).toBe('Hello, World!');
            expect(bytesRead).toBe(buffer.length);
        });

        test('decodes simple element', () => {
            // VNode format: [type:0x01][tag][hid][attrCount][childCount]
            const parts = [
                new Uint8Array([0x01]), // ELEMENT type
                codec.encodeString('div'),
                codec.encodeString('h1'), // hid
                codec.encodeUvarint(0), // 0 attributes
                codec.encodeUvarint(0), // 0 children
            ];

            let totalLength = 0;
            for (const p of parts) totalLength += p.length;
            const buffer = new Uint8Array(totalLength);
            let offset = 0;
            for (const p of parts) {
                buffer.set(p, offset);
                offset += p.length;
            }

            const { vnode } = codec.decodeVNode(buffer, 0);

            expect(vnode.type).toBe('element');
            expect(vnode.tag).toBe('div');
            expect(vnode.hid).toBe('h1');
            expect(Object.keys(vnode.attrs).length).toBe(0);
            expect(vnode.children.length).toBe(0);
        });

        test('decodes element with attributes', () => {
            const parts = [
                new Uint8Array([0x01]), // ELEMENT type
                codec.encodeString('button'),
                codec.encodeString('h2'), // hid
                codec.encodeUvarint(2), // 2 attributes
                codec.encodeString('class'),
                codec.encodeString('btn primary'),
                codec.encodeString('type'),
                codec.encodeString('submit'),
                codec.encodeUvarint(0), // 0 children
            ];

            let totalLength = 0;
            for (const p of parts) totalLength += p.length;
            const buffer = new Uint8Array(totalLength);
            let offset = 0;
            for (const p of parts) {
                buffer.set(p, offset);
                offset += p.length;
            }

            const { vnode } = codec.decodeVNode(buffer, 0);

            expect(vnode.type).toBe('element');
            expect(vnode.tag).toBe('button');
            expect(vnode.attrs.class).toBe('btn primary');
            expect(vnode.attrs.type).toBe('submit');
        });
    });
});

describe('EventType constants', () => {
    test('mouse events are in correct range', () => {
        expect(EventType.CLICK).toBe(0x01);
        expect(EventType.DBLCLICK).toBe(0x02);
        expect(EventType.MOUSEENTER).toBe(0x06);
        expect(EventType.MOUSELEAVE).toBe(0x07);
    });

    test('form events are in correct range', () => {
        expect(EventType.INPUT).toBe(0x10);
        expect(EventType.CHANGE).toBe(0x11);
        expect(EventType.SUBMIT).toBe(0x12);
        expect(EventType.FOCUS).toBe(0x13);
        expect(EventType.BLUR).toBe(0x14);
    });

    test('keyboard events are in correct range', () => {
        expect(EventType.KEYDOWN).toBe(0x20);
        expect(EventType.KEYUP).toBe(0x21);
    });

    test('special events have correct values', () => {
        expect(EventType.HOOK).toBe(0x60);
        expect(EventType.NAVIGATE).toBe(0x70);
    });
});

describe('PatchType constants', () => {
    test('core operations have correct values', () => {
        expect(PatchType.SET_TEXT).toBe(0x01);
        expect(PatchType.SET_ATTR).toBe(0x02);
        expect(PatchType.REMOVE_ATTR).toBe(0x03);
        expect(PatchType.INSERT_NODE).toBe(0x04);
        expect(PatchType.REMOVE_NODE).toBe(0x05);
        expect(PatchType.REPLACE_NODE).toBe(0x07);
    });

    test('class operations have correct values', () => {
        expect(PatchType.ADD_CLASS).toBe(0x10);
        expect(PatchType.REMOVE_CLASS).toBe(0x11);
    });
});
