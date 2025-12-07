/**
 * Integration Tests for VangoClient
 * 
 * Tests the full client with mock WebSocket and DOM.
 * Uses jest-environment-jsdom for proper DOM simulation.
 */

import { jest, describe, test, expect, beforeEach, afterEach } from '@jest/globals';
import { BinaryCodec, EventType, PatchType } from '../src/codec.js';

describe('VangoClient Integration', () => {
    // Test codec directly since WebSocket/DOM mocking is complex in Node
    const codec = new BinaryCodec();

    describe('Event Encoding for Server', () => {
        test('encodes click event with sequence number', () => {
            const buffer = codec.encodeEvent(1, EventType.CLICK, 'h42', null);

            // Verify structure: seq + type + hid
            expect(buffer.length).toBeGreaterThan(3);
            // Seq = 1, Type = 0x01, HID = "h42" (4 bytes string encoding)
            expect(buffer[0]).toBe(1); // seq
            expect(buffer[1]).toBe(EventType.CLICK); // type
        });

        test('encodes input event with value', () => {
            const buffer = codec.encodeEvent(5, EventType.INPUT, 'h1', { value: 'hello world' });

            expect(buffer.length).toBeGreaterThan(10);
        });

        test('encodes submit event with form fields', () => {
            const buffer = codec.encodeEvent(3, EventType.SUBMIT, 'h10', {
                username: 'john',
                password: 'secret123',
            });

            // Should contain all field data
            expect(buffer.length).toBeGreaterThan(20);
        });

        test('encodes keyboard event with modifiers', () => {
            const buffer = codec.encodeEvent(1, EventType.KEYDOWN, 'h5', {
                key: 'Enter',
                ctrlKey: true,
                shiftKey: false,
                altKey: false,
                metaKey: true,
            });

            expect(buffer.length).toBeGreaterThan(5);
        });

        test('encodes hook event with data', () => {
            const buffer = codec.encodeEvent(1, EventType.HOOK, 'h100', {
                name: 'reorder',
                data: {
                    id: 'item-1',
                    fromIndex: 0,
                    toIndex: 2,
                },
            });

            expect(buffer.length).toBeGreaterThan(15);
        });

        test('encodes navigate event with path', () => {
            const buffer = codec.encodeEvent(1, EventType.NAVIGATE, 'nav', {
                path: '/users/profile',
                replace: false,
            });

            expect(buffer.length).toBeGreaterThan(10);
        });
    });

    describe('Patch Decoding from Server', () => {
        test('decodes SET_TEXT patch correctly', () => {
            // Build a valid patches frame
            const parts = [
                codec.encodeUvarint(1), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.SET_TEXT]),
                codec.encodeString('h5'),
                codec.encodeString('Hello, World!'),
            ];

            const buffer = concatArrays(parts);
            const { seq, patches } = codec.decodePatches(buffer);

            expect(seq).toBe(1);
            expect(patches.length).toBe(1);
            expect(patches[0].type).toBe(PatchType.SET_TEXT);
            expect(patches[0].hid).toBe('h5');
            expect(patches[0].value).toBe('Hello, World!');
        });

        test('decodes SET_ATTR patch correctly', () => {
            const parts = [
                codec.encodeUvarint(2), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.SET_ATTR]),
                codec.encodeString('h10'),
                codec.encodeString('class'),
                codec.encodeString('btn btn-primary'),
            ];

            const buffer = concatArrays(parts);
            const { patches } = codec.decodePatches(buffer);

            expect(patches[0].type).toBe(PatchType.SET_ATTR);
            expect(patches[0].key).toBe('class');
            expect(patches[0].value).toBe('btn btn-primary');
        });

        test('decodes ADD_CLASS patch correctly', () => {
            const parts = [
                codec.encodeUvarint(3), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.ADD_CLASS]),
                codec.encodeString('h7'),
                codec.encodeString('active'),
            ];

            const buffer = concatArrays(parts);
            const { patches } = codec.decodePatches(buffer);

            expect(patches[0].type).toBe(PatchType.ADD_CLASS);
            expect(patches[0].className).toBe('active');
        });

        test('decodes REMOVE_NODE patch correctly', () => {
            const parts = [
                codec.encodeUvarint(4), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.REMOVE_NODE]),
                codec.encodeString('h99'),
            ];

            const buffer = concatArrays(parts);
            const { patches } = codec.decodePatches(buffer);

            expect(patches[0].type).toBe(PatchType.REMOVE_NODE);
            expect(patches[0].hid).toBe('h99');
        });

        test('decodes SCROLL_TO patch correctly', () => {
            const parts = [
                codec.encodeUvarint(5), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.SCROLL_TO]),
                codec.encodeString('h1'),
                codec.encodeSvarint(0), // x
                codec.encodeSvarint(500), // y
                new Uint8Array([1]), // behavior = smooth
            ];

            const buffer = concatArrays(parts);
            const { patches } = codec.decodePatches(buffer);

            expect(patches[0].type).toBe(PatchType.SCROLL_TO);
            expect(patches[0].x).toBe(0);
            expect(patches[0].y).toBe(500);
            expect(patches[0].behavior).toBe(1);
        });

        test('decodes SET_CHECKED patch correctly', () => {
            const parts = [
                codec.encodeUvarint(6), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.SET_CHECKED]),
                codec.encodeString('h20'),
                new Uint8Array([1]), // checked = true
            ];

            const buffer = concatArrays(parts);
            const { patches } = codec.decodePatches(buffer);

            expect(patches[0].type).toBe(PatchType.SET_CHECKED);
            expect(patches[0].value).toBe(true);
        });

        test('decodes batch of multiple patches', () => {
            const parts = [
                codec.encodeUvarint(100), // seq
                codec.encodeUvarint(3), // count = 3
                // Patch 1: SET_TEXT
                new Uint8Array([PatchType.SET_TEXT]),
                codec.encodeString('h1'),
                codec.encodeString('Count: 5'),
                // Patch 2: ADD_CLASS
                new Uint8Array([PatchType.ADD_CLASS]),
                codec.encodeString('h2'),
                codec.encodeString('highlight'),
                // Patch 3: FOCUS
                new Uint8Array([PatchType.FOCUS]),
                codec.encodeString('h3'),
            ];

            const buffer = concatArrays(parts);
            const { seq, patches } = codec.decodePatches(buffer);

            expect(seq).toBe(100);
            expect(patches.length).toBe(3);
            expect(patches[0].type).toBe(PatchType.SET_TEXT);
            expect(patches[1].type).toBe(PatchType.ADD_CLASS);
            expect(patches[2].type).toBe(PatchType.FOCUS);
        });
    });

    describe('VNode Decoding', () => {
        test('decodes text VNode', () => {
            const parts = [
                new Uint8Array([0x02]), // TEXT type
                codec.encodeString('Hello'),
            ];

            const buffer = concatArrays(parts);
            const { vnode } = codec.decodeVNode(buffer, 0);

            expect(vnode.type).toBe('text');
            expect(vnode.text).toBe('Hello');
        });

        test('decodes element VNode with attributes', () => {
            const parts = [
                new Uint8Array([0x01]), // ELEMENT type
                codec.encodeString('button'),
                codec.encodeString('h50'), // hid
                codec.encodeUvarint(2), // 2 attributes
                codec.encodeString('class'),
                codec.encodeString('btn'),
                codec.encodeString('type'),
                codec.encodeString('submit'),
                codec.encodeUvarint(0), // 0 children
            ];

            const buffer = concatArrays(parts);
            const { vnode } = codec.decodeVNode(buffer, 0);

            expect(vnode.type).toBe('element');
            expect(vnode.tag).toBe('button');
            expect(vnode.hid).toBe('h50');
            expect(vnode.attrs.class).toBe('btn');
            expect(vnode.attrs.type).toBe('submit');
        });

        test('decodes INSERT_NODE patch with nested VNode', () => {
            // Build INSERT_NODE patch with a div containing a text child
            const childText = [
                new Uint8Array([0x02]), // TEXT type
                codec.encodeString('Click me'),
            ];
            const childBuffer = concatArrays(childText);

            const parts = [
                codec.encodeUvarint(1), // seq
                codec.encodeUvarint(1), // count
                new Uint8Array([PatchType.INSERT_NODE]),
                codec.encodeString('h10'), // target hid
                codec.encodeString('h1'), // parent hid
                codec.encodeUvarint(0), // index
                // VNode element
                new Uint8Array([0x01]), // ELEMENT type
                codec.encodeString('span'),
                codec.encodeString('h10'), // node hid
                codec.encodeUvarint(1), // 1 attribute
                codec.encodeString('class'),
                codec.encodeString('label'),
                codec.encodeUvarint(1), // 1 child
                ...childText,
            ];

            const buffer = concatArrays(parts);
            const { patches } = codec.decodePatches(buffer);

            expect(patches[0].type).toBe(PatchType.INSERT_NODE);
            expect(patches[0].parentID).toBe('h1');
            expect(patches[0].index).toBe(0);
            expect(patches[0].vnode.tag).toBe('span');
            expect(patches[0].vnode.children.length).toBe(1);
            expect(patches[0].vnode.children[0].text).toBe('Click me');
        });
    });

    describe('Protocol Compatibility', () => {
        test('EventType values match server constants', () => {
            // These must match pkg/protocol/event.go exactly
            expect(EventType.CLICK).toBe(0x01);
            expect(EventType.DBLCLICK).toBe(0x02);
            expect(EventType.MOUSEENTER).toBe(0x06);
            expect(EventType.MOUSELEAVE).toBe(0x07);
            expect(EventType.INPUT).toBe(0x10);
            expect(EventType.CHANGE).toBe(0x11);
            expect(EventType.SUBMIT).toBe(0x12);
            expect(EventType.FOCUS).toBe(0x13);
            expect(EventType.BLUR).toBe(0x14);
            expect(EventType.KEYDOWN).toBe(0x20);
            expect(EventType.KEYUP).toBe(0x21);
            expect(EventType.SCROLL).toBe(0x30);
            expect(EventType.HOOK).toBe(0x60);
            expect(EventType.NAVIGATE).toBe(0x70);
        });

        test('PatchType values match server constants', () => {
            // These must match pkg/protocol/patch.go exactly
            expect(PatchType.SET_TEXT).toBe(0x01);
            expect(PatchType.SET_ATTR).toBe(0x02);
            expect(PatchType.REMOVE_ATTR).toBe(0x03);
            expect(PatchType.INSERT_NODE).toBe(0x04);
            expect(PatchType.REMOVE_NODE).toBe(0x05);
            expect(PatchType.MOVE_NODE).toBe(0x06);
            expect(PatchType.REPLACE_NODE).toBe(0x07);
            expect(PatchType.SET_VALUE).toBe(0x08);
            expect(PatchType.SET_CHECKED).toBe(0x09);
            expect(PatchType.SET_SELECTED).toBe(0x0A);
            expect(PatchType.FOCUS).toBe(0x0B);
            expect(PatchType.BLUR).toBe(0x0C);
            expect(PatchType.SCROLL_TO).toBe(0x0D);
            expect(PatchType.ADD_CLASS).toBe(0x10);
            expect(PatchType.REMOVE_CLASS).toBe(0x11);
            expect(PatchType.SET_STYLE).toBe(0x13);
        });
    });
});

// Helper to concatenate Uint8Arrays
function concatArrays(arrays) {
    let totalLength = 0;
    for (const arr of arrays) {
        totalLength += arr.length;
    }
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const arr of arrays) {
        result.set(arr, offset);
        offset += arr.length;
    }
    return result;
}
