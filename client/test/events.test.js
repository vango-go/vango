/**
 * Event Capture Tests
 *
 * Tests for the interception decision table and progressive enhancement
 * per spec Section 5.2.
 */

import { jest, describe, test, expect, beforeEach } from '@jest/globals';
import { EventCapture } from '../src/events.js';

/**
 * Create a mock click event with configurable properties
 */
function createClickEvent(options = {}) {
    const event = {
        type: 'click',
        target: options.target || document.createElement('div'),
        button: options.button ?? 0,
        ctrlKey: options.ctrlKey ?? false,
        metaKey: options.metaKey ?? false,
        shiftKey: options.shiftKey ?? false,
        altKey: options.altKey ?? false,
        defaultPrevented: options.defaultPrevented ?? false,
        preventDefault: jest.fn(),
        stopPropagation: jest.fn(),
    };
    return event;
}

/**
 * Create a mock submit event
 */
function createSubmitEvent(options = {}) {
    const event = {
        type: 'submit',
        target: options.target || document.createElement('form'),
        defaultPrevented: options.defaultPrevented ?? false,
        preventDefault: jest.fn(),
        stopPropagation: jest.fn(),
    };
    // Add closest method to target for form detection
    if (!event.target.closest) {
        event.target.closest = (selector) => {
            if (selector === 'form[data-hid]' && event.target.matches && event.target.matches(selector)) {
                return event.target;
            }
            return null;
        };
    }
    return event;
}

/**
 * Create a mock client with configurable connected state
 */
function createMockClient(connected = true) {
    return {
        connected,
        options: { debug: false },
        optimistic: {
            applyOptimistic: jest.fn(),
        },
        sendEvent: jest.fn(),
    };
}

/**
 * Create an element with data-hid and data-ve attributes
 */
function createHidElement(hid, events = 'click') {
    const el = document.createElement('div');
    el.dataset.hid = hid;
    el.dataset.ve = events;
    return el;
}

describe('EventCapture', () => {
    let eventCapture;
    let client;

    beforeEach(() => {
        client = createMockClient(true);
        eventCapture = new EventCapture(client);
        document.body.innerHTML = '';
    });

    describe('click interception decision table', () => {
        test('does not intercept if event.defaultPrevented', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({
                target: el,
                defaultPrevented: true,
            });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
            expect(event.preventDefault).not.toHaveBeenCalled();
        });

        test('does not intercept right-click (button !== 0)', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({
                target: el,
                button: 2, // Right click
            });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept middle-click (button === 1)', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({
                target: el,
                button: 1, // Middle click
            });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept ctrl+click', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({
                target: el,
                ctrlKey: true,
            });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept meta+click', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({
                target: el,
                metaKey: true,
            });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept shift+click', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({
                target: el,
                shiftKey: true,
            });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept alt+click', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({
                target: el,
                altKey: true,
            });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept when WS disconnected', () => {
            client.connected = false;

            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({ target: el });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept anchor with target="_blank"', () => {
            const anchor = document.createElement('a');
            anchor.href = '/page';
            anchor.target = '_blank';
            anchor.dataset.hid = 'h1';
            anchor.dataset.ve = 'click';
            document.body.appendChild(anchor);

            const event = createClickEvent({ target: anchor });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept anchor with target="_top"', () => {
            const anchor = document.createElement('a');
            anchor.href = '/page';
            anchor.target = '_top';
            anchor.dataset.hid = 'h1';
            anchor.dataset.ve = 'click';
            document.body.appendChild(anchor);

            const event = createClickEvent({ target: anchor });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept anchor with download attribute', () => {
            const anchor = document.createElement('a');
            anchor.href = '/file.pdf';
            anchor.setAttribute('download', '');
            anchor.dataset.hid = 'h1';
            anchor.dataset.ve = 'click';
            document.body.appendChild(anchor);

            const event = createClickEvent({ target: anchor });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('does not intercept cross-origin anchor', () => {
            const anchor = document.createElement('a');
            anchor.href = 'https://external.com/page';
            anchor.dataset.hid = 'h1';
            anchor.dataset.ve = 'click';
            document.body.appendChild(anchor);

            const event = createClickEvent({ target: anchor });

            eventCapture._handleClick(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('intercepts valid left-click on data-ve element', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const event = createClickEvent({ target: el });

            eventCapture._handleClick(event);

            expect(client.sendEvent).toHaveBeenCalledWith(expect.anything(), 'h1');
            expect(event.preventDefault).toHaveBeenCalled();
        });

        test('intercepts anchor with target="_self"', () => {
            const anchor = document.createElement('a');
            anchor.href = '/page';
            anchor.target = '_self';
            anchor.dataset.hid = 'h1';
            anchor.dataset.ve = 'click';
            document.body.appendChild(anchor);

            const event = createClickEvent({ target: anchor });

            eventCapture._handleClick(event);

            expect(client.sendEvent).toHaveBeenCalled();
            expect(event.preventDefault).toHaveBeenCalled();
        });

        test('intercepts click on child of element with data-ve', () => {
            const parent = createHidElement('h1', 'click');
            const child = document.createElement('span');
            child.textContent = 'Click me';
            parent.appendChild(child);
            document.body.appendChild(parent);

            const event = createClickEvent({ target: child });

            eventCapture._handleClick(event);

            expect(client.sendEvent).toHaveBeenCalledWith(expect.anything(), 'h1');
        });

        test('checks anchor via event.target.closest, not just the HID element', () => {
            // Create: <a href="/page"><div data-hid="h1" data-ve="click">text</div></a>
            const anchor = document.createElement('a');
            anchor.href = 'https://external.com/page'; // Cross-origin
            const inner = createHidElement('h1', 'click');
            anchor.appendChild(inner);
            document.body.appendChild(anchor);

            const event = createClickEvent({ target: inner });

            eventCapture._handleClick(event);

            // Should NOT intercept because the click is inside a cross-origin anchor
            expect(client.sendEvent).not.toHaveBeenCalled();
        });
    });

    describe('submit progressive enhancement', () => {
        test('does not intercept submit when WS disconnected', () => {
            client.connected = false;

            const form = document.createElement('form');
            form.dataset.hid = 'f1';
            form.dataset.ve = 'submit';
            document.body.appendChild(form);

            const event = createSubmitEvent({ target: form });

            eventCapture._handleSubmit(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
            // Most importantly: preventDefault should NOT be called
            expect(event.preventDefault).not.toHaveBeenCalled();
        });

        test('intercepts submit when WS connected', () => {
            const form = document.createElement('form');
            form.dataset.hid = 'f1';
            form.dataset.ve = 'submit';
            document.body.appendChild(form);

            const event = createSubmitEvent({ target: form });

            eventCapture._handleSubmit(event);

            expect(client.sendEvent).toHaveBeenCalled();
            expect(event.preventDefault).toHaveBeenCalled();
        });

        test('does not intercept submit if already defaultPrevented', () => {
            const form = document.createElement('form');
            form.dataset.hid = 'f1';
            form.dataset.ve = 'submit';
            document.body.appendChild(form);

            const event = createSubmitEvent({
                target: form,
                defaultPrevented: true,
            });

            eventCapture._handleSubmit(event);

            expect(client.sendEvent).not.toHaveBeenCalled();
        });

        test('collects form data and sends to server', () => {
            const form = document.createElement('form');
            form.dataset.hid = 'f1';
            form.dataset.ve = 'submit';

            const input = document.createElement('input');
            input.name = 'username';
            input.value = 'testuser';
            form.appendChild(input);

            document.body.appendChild(form);

            const event = createSubmitEvent({ target: form });

            eventCapture._handleSubmit(event);

            expect(client.sendEvent).toHaveBeenCalledWith(
                expect.anything(),
                'f1',
                expect.objectContaining({ username: 'testuser' })
            );
        });
    });

    describe('_findHidElementWithEvent', () => {
        test('returns null if target has no closest method', () => {
            const result = eventCapture._findHidElementWithEvent(null, 'click');
            expect(result).toBeNull();
        });

        test('returns null if no HID element found', () => {
            const el = document.createElement('div');
            document.body.appendChild(el);

            const result = eventCapture._findHidElementWithEvent(el, 'click');
            expect(result).toBeNull();
        });

        test('finds element with matching event in data-ve', () => {
            const el = createHidElement('h1', 'click');
            document.body.appendChild(el);

            const result = eventCapture._findHidElementWithEvent(el, 'click');
            expect(result).toBe(el);
        });

        test('bubbles up to find parent with matching event', () => {
            const parent = createHidElement('h1', 'click');
            const child = document.createElement('span');
            parent.appendChild(child);
            document.body.appendChild(parent);

            const result = eventCapture._findHidElementWithEvent(child, 'click');
            expect(result).toBe(parent);
        });

        test('skips HID elements without the matching event', () => {
            const outer = createHidElement('h1', 'click');
            const inner = createHidElement('h2', 'input'); // Different event
            const deepChild = document.createElement('span');

            inner.appendChild(deepChild);
            outer.appendChild(inner);
            document.body.appendChild(outer);

            const result = eventCapture._findHidElementWithEvent(deepChild, 'click');
            expect(result).toBe(outer);
        });
    });

    describe('_hasEvent', () => {
        test('returns true for single event match', () => {
            const el = createHidElement('h1', 'click');
            expect(eventCapture._hasEvent(el, 'click')).toBe(true);
        });

        test('returns true for event in comma-separated list', () => {
            const el = createHidElement('h1', 'click,input,change');
            expect(eventCapture._hasEvent(el, 'input')).toBe(true);
        });

        test('returns false for non-matching event', () => {
            const el = createHidElement('h1', 'click');
            expect(eventCapture._hasEvent(el, 'input')).toBe(false);
        });

        test('handles whitespace in data-ve', () => {
            const el = createHidElement('h1', 'click, input, change');
            expect(eventCapture._hasEvent(el, 'input')).toBe(true);
        });
    });
});
