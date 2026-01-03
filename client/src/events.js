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

        event.preventDefault();

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

        event.preventDefault();
        this.client.sendEvent(EventType.DBLCLICK, el.dataset.hid);
    }

    /**
     * Handle input event (debounced)
     */
    _handleInput(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'input')) return;

        const hid = el.dataset.hid;
        const debounceMs = parseInt(el.dataset.debounce || '100', 10);

        // Clear existing timer
        if (this.debounceTimers.has(hid)) {
            clearTimeout(this.debounceTimers.get(hid));
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

        this.client.sendEvent(EventType.FOCUS, el.dataset.hid);
    }

    /**
     * Handle blur event
     */
    _handleBlur(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'blur')) return;

        this.client.sendEvent(EventType.BLUR, el.dataset.hid);
    }

    /**
     * Handle keydown event
     */
    _handleKeyDown(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'keydown')) return;

        // Check for specific key filter
        const keyFilter = el.dataset.keyFilter;
        if (keyFilter && !this._matchesKeyFilter(event, keyFilter)) {
            return;
        }

        // Some key combinations should not prevent default
        const shouldPrevent = el.dataset.preventDefault !== 'false';
        if (shouldPrevent) {
            event.preventDefault();
        }

        this.client.sendEvent(EventType.KEYDOWN, el.dataset.hid, {
            key: event.key,
            code: event.code,
            ctrlKey: event.ctrlKey,
            shiftKey: event.shiftKey,
            altKey: event.altKey,
            metaKey: event.metaKey,
        });
    }

    /**
     * Handle keyup event
     */
    _handleKeyUp(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'keyup')) return;

        this.client.sendEvent(EventType.KEYUP, el.dataset.hid, {
            key: event.key,
            code: event.code,
            ctrlKey: event.ctrlKey,
            shiftKey: event.shiftKey,
            altKey: event.altKey,
            metaKey: event.metaKey,
        });
    }

    /**
     * Handle mouseenter event
     */
    _handleMouseEnter(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'mouseenter')) return;

        this.client.sendEvent(EventType.MOUSEENTER, el.dataset.hid);
    }

    /**
     * Handle mouseleave event
     */
    _handleMouseLeave(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'mouseleave')) return;

        this.client.sendEvent(EventType.MOUSELEAVE, el.dataset.hid);
    }

    /**
     * Handle scroll event (throttled)
     */
    _handleScroll(event) {
        const el = this._findHidElement(event.target);
        if (!el || !this._hasEvent(el, 'scroll')) return;

        const hid = el.dataset.hid;
        const throttleMs = parseInt(el.dataset.throttle || '100', 10);

        // Simple throttle
        if (this.scrollThrottled.has(hid)) {
            return;
        }

        this.scrollThrottled.add(hid);

        setTimeout(() => {
            this.scrollThrottled.delete(hid);
        }, throttleMs);

        this.client.sendEvent(EventType.SCROLL, hid, {
            scrollTop: el.scrollTop,
            scrollLeft: el.scrollLeft,
        });
    }

    /**
     * Handle link click for client-side navigation.
     *
     * Per spec Section 3.2 and 9.5.1, do NOT intercept when:
     * - Element is not an <a> tag
     * - Any modifier key is held (Ctrl/Meta/Shift/Alt)
     * - Link has download attribute
     * - Link has target != "" && target != "_self"
     * - Link URL is cross-origin
     * - WebSocket not connected/healthy
     * - No Vango handler exists (data-vango-link or data-ve="click")
     *
     * Otherwise: preventDefault and send NAVIGATE event.
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
        // Supports both data-link (from router.Link) and data-vango-link (from vdom.NavLink)
        const isVangoLink = link.hasAttribute('data-vango-link') || link.hasAttribute('data-link');
        const hasClickHandler = this._hasEvent(link, 'click');

        // Only intercept Vango-marked links or links with click handlers
        if (!isVangoLink && !hasClickHandler) {
            return; // Let browser handle native navigation
        }

        // Check if this is an internal link (same origin)
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

        // Prevent default and navigate via WebSocket
        event.preventDefault();

        // Update browser URL
        history.pushState(null, '', href);

        // Send navigate event to server
        this.client.sendEvent(EventType.NAVIGATE, 'nav', { path: href });
    }

    /**
     * Handle browser back/forward navigation.
     * Sends the full path including search params.
     */
    _handlePopState(event) {
        // Include search params in navigation path
        const path = location.pathname + location.search;
        this.client.sendEvent(EventType.NAVIGATE, 'nav', { path });
    }

    /**
     * Handle prefetch on hover for links with data-prefetch attribute.
     * Per spec Section 9.4: "Preloads on hover"
     *
     * This sends a prefetch event to the server which can preload the target
     * page's data before the user clicks, making navigation feel instant.
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
        // The server can use this to preload route data
        this.client.sendEvent(EventType.CUSTOM, 'prefetch', { name: 'prefetch', path: href });

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
