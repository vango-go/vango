/**
 * Event Capture and Delegation
 *
 * Captures user events on elements with data-hid attributes and sends them
 * to the server via WebSocket.
 */

import { EventType } from './codec.js';

export class EventCapture {
    constructor(client) {
        this.client = client;
        this.handlers = new Map();
        this.debounceTimers = new Map();
        this.scrollThrottled = new Set();
        this.prefetchedPaths = new Set(); // Track prefetched paths to avoid duplicates
    }

    /**
     * Attach event listeners to document
     */
    attach() {
        // Click events
        this._on('click', this._handleClick.bind(this));
        this._on('dblclick', this._handleDblClick.bind(this));

        // Input events (debounced)
        this._on('input', this._handleInput.bind(this));
        this._on('change', this._handleChange.bind(this));

        // Form events
        this._on('submit', this._handleSubmit.bind(this));

        // Focus events (capture phase for delegation)
        this._on('focus', this._handleFocus.bind(this), true);
        this._on('blur', this._handleBlur.bind(this), true);

        // Keyboard events
        this._on('keydown', this._handleKeyDown.bind(this));
        this._on('keyup', this._handleKeyUp.bind(this));

        // Mouse events
        this._on('mouseenter', this._handleMouseEnter.bind(this), true);
        this._on('mouseleave', this._handleMouseLeave.bind(this), true);

        // Scroll events (throttled)
        this._on('scroll', this._handleScroll.bind(this), true);

        // Navigation
        this._on('click', this._handleLinkClick.bind(this));
        window.addEventListener('popstate', this._handlePopState.bind(this));

        // Prefetch on hover for links with data-prefetch attribute
        this._on('mouseenter', this._handlePrefetch.bind(this), true);
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
        // Handle non-element targets (document, text nodes, etc.)
        if (!target || !target.closest) {
            return null;
        }
        return target.closest('[data-hid]');
    }

    /**
     * Check if element has an event in its data-ve attribute.
     * Parses data-ve="click,input,change" format per spec Section 5.2.
     */
    _hasEvent(el, eventName) {
        const ve = (el.dataset.ve || '').split(',').map(s => s.trim());
        return ve.includes(eventName);
    }

    /**
     * Get modifier options for an event on an element.
     * Reads data-pd-{event}, data-sp-{event}, data-self-{event}, etc.
     * Per spec section 3.9.3, lines 1299-1365.
     */
    _getModifiers(el, eventName) {
        return {
            preventDefault: el.dataset[`pd${this._capitalize(eventName)}`] === 'true',
            stopPropagation: el.dataset[`sp${this._capitalize(eventName)}`] === 'true',
            self: el.dataset[`self${this._capitalize(eventName)}`] === 'true',
            once: el.dataset[`once${this._capitalize(eventName)}`] === 'true',
            passive: el.dataset[`passive${this._capitalize(eventName)}`] === 'true',
            capture: el.dataset[`capture${this._capitalize(eventName)}`] === 'true',
            debounce: parseInt(el.dataset[`debounce${this._capitalize(eventName)}`] || el.dataset.debounce || '0', 10),
            throttle: parseInt(el.dataset[`throttle${this._capitalize(eventName)}`] || el.dataset.throttle || '0', 10),
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

        // Self modifier - only fire if target is the exact element
        if (mods.self && event.target !== el) {
            return false;
        }

        // Passive modifier - per spec section 3.9.3 lines 1329-1332:
        // "Passive handlers cannot call preventDefault"
        // We enforce this by NOT calling preventDefault when passive is set,
        // even if the handler or other modifiers would normally call it.
        // Note: With event delegation, we cannot truly register as passive at the
        // browser level per-element, but we enforce the semantic constraint.
        if (mods.passive) {
            // Store original preventDefault to block it
            const originalPreventDefault = event.preventDefault.bind(event);
            event.preventDefault = () => {
                console.warn('[Vango] preventDefault() called on passive handler - ignored');
            };
            // Restore after microtask to not affect other handlers
            queueMicrotask(() => {
                event.preventDefault = originalPreventDefault;
            });
        } else if (mods.preventDefault) {
            // PreventDefault modifier - only if not passive
            event.preventDefault();
        }

        // StopPropagation modifier
        if (mods.stopPropagation) {
            event.stopPropagation();
        }

        // Once modifier - remove event from data-ve after first trigger
        if (mods.once) {
            const ve = (el.dataset.ve || '').split(',').map(s => s.trim());
            const filtered = ve.filter(e => e !== eventName);
            if (filtered.length > 0) {
                el.dataset.ve = filtered.join(',');
            } else {
                delete el.dataset.ve;
            }
        }

        // Capture modifier note: With event delegation at the document level,
        // true per-element capture phase handling is not possible. The capture
        // flag is available for documentation/future use, but currently some
        // events (focus, blur, scroll, mouseenter, mouseleave) are always
        // captured, while others (click, input, keydown) bubble.

        return true;
    }

    /**
     * Get debounce delay for an event on an element.
     */
    _getDebounce(el, eventName) {
        const mods = this._getModifiers(el, eventName);
        // Fall back to default for input events
        if (eventName === 'input' && mods.debounce === 0) {
            return 100;
        }
        return mods.debounce;
    }

    /**
     * Get throttle delay for an event on an element.
     */
    _getThrottle(el, eventName) {
        const mods = this._getModifiers(el, eventName);
        // Fall back to default for scroll events
        if (eventName === 'scroll' && mods.throttle === 0) {
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
        // Handle non-element targets
        if (!target || !target.closest) {
            return null;
        }

        // Start from the target and traverse up
        let el = target.closest('[data-hid]');
        while (el) {
            if (this._hasEvent(el, eventName)) {
                return el;
            }
            // Move to parent and find next HID element
            const parent = el.parentElement;
            if (!parent) break;
            el = parent.closest('[data-hid]');
        }
        return null;
    }

    /**
     * Handle click event
     */
    _handleClick(event) {
        // Find the closest HID element with a click event in data-ve, bubbling up through ancestors
        const el = this._findHidElementWithEvent(event.target, 'click');
        if (!el) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'click')) {
            return;
        }

        // Default preventDefault for click events (unless passive)
        const mods = this._getModifiers(el, 'click');
        if (!mods.preventDefault && !mods.passive) {
            event.preventDefault();
        }

        // Apply optimistic updates if configured
        this.client.optimistic.applyOptimistic(el, 'click');

        // Send event
        this.client.sendEvent(EventType.CLICK, el.dataset.hid);
    }

    /**
     * Handle double-click event
     */
    _handleDblClick(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'dblclick')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'dblclick')) {
            return;
        }

        // Default preventDefault for dblclick events (unless passive)
        const mods = this._getModifiers(el, 'dblclick');
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
        if (!el || !this._hasEvent(el, 'input')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'input')) {
            return;
        }

        const hid = el.dataset.hid;
        const debounceMs = this._getDebounce(el, 'input');

        // Clear existing timer
        if (this.debounceTimers.has(hid)) {
            clearTimeout(this.debounceTimers.get(hid));
        }

        // If debounce is 0, send immediately
        if (debounceMs === 0) {
            this.client.sendEvent(EventType.INPUT, hid, { value: el.value });
            return;
        }

        // Set new timer
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
        if (!el || !this._hasEvent(el, 'change')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'change')) {
            return;
        }

        let value;
        if (el.type === 'checkbox') {
            value = el.checked ? 'true' : 'false';
        } else if (el.type === 'radio') {
            value = el.checked ? el.value : '';
        } else if (el.tagName === 'SELECT' && el.multiple) {
            value = Array.from(el.selectedOptions).map(o => o.value).join(',');
        } else {
            value = el.value;
        }

        // Apply optimistic updates
        this.client.optimistic.applyOptimistic(el, 'change');

        this.client.sendEvent(EventType.CHANGE, el.dataset.hid, { value });
    }

    /**
     * Handle form submit
     */
    _handleSubmit(event) {
        const form = event.target.closest('form[data-hid]');
        if (!form || !this._hasEvent(form, 'submit')) return;

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
        if (!el || !this._hasEvent(el, 'focus')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'focus')) {
            return;
        }

        this.client.sendEvent(EventType.FOCUS, el.dataset.hid);
    }

    /**
     * Handle blur event
     */
    _handleBlur(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'blur')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'blur')) {
            return;
        }

        this.client.sendEvent(EventType.BLUR, el.dataset.hid);
    }

    /**
     * Handle keydown event
     */
    _handleKeyDown(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'keydown')) return;

        // Check for specific key filter (legacy support)
        const keyFilter = el.dataset.keyFilter;
        if (keyFilter && !this._matchesKeyFilter(event, keyFilter)) {
            return;
        }

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'keydown')) {
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
            location: event.location,
        });
    }

    /**
     * Handle keyup event
     */
    _handleKeyUp(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'keyup')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'keyup')) {
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
            location: event.location,
        });
    }

    /**
     * Handle mouseenter event
     */
    _handleMouseEnter(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'mouseenter')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'mouseenter')) {
            return;
        }

        this.client.sendEvent(EventType.MOUSEENTER, el.dataset.hid);
    }

    /**
     * Handle mouseleave event
     */
    _handleMouseLeave(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'mouseleave')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'mouseleave')) {
            return;
        }

        this.client.sendEvent(EventType.MOUSELEAVE, el.dataset.hid);
    }

    /**
     * Handle scroll event (throttled)
     */
    _handleScroll(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'scroll')) return;

        // Apply modifiers (may skip if Self modifier fails)
        if (!this._applyModifiers(event, el, 'scroll')) {
            return;
        }

        const hid = el.dataset.hid;
        const throttleMs = this._getThrottle(el, 'scroll');

        // Simple throttle (skip if throttle is 0 for immediate dispatch)
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
            scrollLeft: el.scrollLeft,
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
        const link = event.target.closest('a[href]');
        if (!link) return;

        // Don't intercept if modifier keys are held (allow open in new tab, etc.)
        if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
            return;
        }

        // Don't intercept download links
        if (link.hasAttribute('download')) {
            return;
        }

        // Don't intercept links with target (except _self)
        const target = link.getAttribute('target');
        if (target && target !== '_self') {
            return;
        }

        // Check if link is marked for SPA navigation
        // data-vango-link is the canonical marker per the routing contract
        // data-link is deprecated but supported for backwards compatibility
        const isVangoLink = link.hasAttribute('data-vango-link') || link.hasAttribute('data-link');

        // Only intercept Vango-marked links
        // Note: Links with click handlers in data-ve are handled by _handleClick, not here
        if (!isVangoLink) {
            return; // Let browser handle native navigation
        }

        // Get the href attribute (raw, may be percent-encoded)
        const href = link.getAttribute('href');
        if (!href) return;

        // Check for cross-origin URLs
        if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('//')) {
            try {
                const url = new URL(href, window.location.origin);
                if (url.origin !== window.location.origin) {
                    return; // Cross-origin, let browser handle
                }
            } catch {
                return; // Invalid URL, let browser handle
            }
        }

        // Don't intercept if explicitly marked external
        if (link.hasAttribute('data-external')) {
            return;
        }

        // Don't intercept if WebSocket is not connected
        if (!this.client.connected) {
            return; // Fall back to native navigation
        }

        // Prevent default browser navigation
        event.preventDefault();

        // Track pending navigation for self-heal (connection loss during nav)
        this.pendingNavPath = href;

        // Per Navigation Contract Section 4.5:
        // Do NOT update history immediately - wait for server NAV_* response
        // Send navigate event to server (server will respond with NAV_* patch + DOM patches)
        this.client.sendEvent(EventType.NAVIGATE, 'nav', { path: href, replace: false });
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
        // Don't handle if WebSocket is not connected
        if (!this.client.connected) {
            return; // Let native navigation take over
        }

        // Include search params in navigation path
        // Note: location.pathname is already decoded by the browser
        const path = location.pathname + location.search;

        // Track pending navigation for self-heal
        this.pendingNavPath = path;

        // Send navigate event with replace: true (URL already changed)
        this.client.sendEvent(EventType.NAVIGATE, 'nav', { path, replace: true });
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
        const link = event.target.closest('a[data-prefetch][href]');
        if (!link) return;

        const href = link.getAttribute('href');
        if (!href) return;

        // Skip external links
        if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('//')) {
            try {
                const url = new URL(href, window.location.origin);
                if (url.origin !== window.location.origin) {
                    return; // Cross-origin, don't prefetch
                }
            } catch {
                return;
            }
        }

        // Skip if already prefetched
        if (this.prefetchedPaths.has(href)) {
            return;
        }

        // Skip if WebSocket not connected
        if (!this.client.connected) {
            return;
        }

        // Mark as prefetched
        this.prefetchedPaths.add(href);

        // Send prefetch event to server
        // Per the contract, data MUST be JSON bytes, not an object
        // The codec's encodeEvent expects data as bytes for CUSTOM events
        const jsonData = JSON.stringify({ path: href });
        const encoder = new TextEncoder();
        const dataBytes = encoder.encode(jsonData);

        this.client.sendEvent(EventType.CUSTOM, 'prefetch', { name: 'prefetch', data: dataBytes });

        if (this.client.options.debug) {
            console.log('[Vango] Prefetching:', href);
        }
    }

    /**
     * Check if event matches key filter
     * Format: "Enter" or "Ctrl+s" or "Meta+Enter"
     */
    _matchesKeyFilter(event, filter) {
        const parts = filter.split('+');
        const key = parts.pop().toLowerCase();
        const modifiers = new Set(parts.map(m => m.toLowerCase()));

        // Check key
        if (event.key.toLowerCase() !== key) {
            return false;
        }

        // Check modifiers
        const hasCtrl = modifiers.has('ctrl') || modifiers.has('control');
        const hasShift = modifiers.has('shift');
        const hasAlt = modifiers.has('alt');
        const hasMeta = modifiers.has('meta') || modifiers.has('cmd');

        if (hasCtrl !== event.ctrlKey) return false;
        if (hasShift !== event.shiftKey) return false;
        if (hasAlt !== event.altKey) return false;
        if (hasMeta !== event.metaKey) return false;

        return true;
    }
}
