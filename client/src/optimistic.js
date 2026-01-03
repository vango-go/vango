/**
 * Optimistic Update Handling
 *
 * Provides instant feedback for user actions before server confirms.
 */

export class OptimisticUpdates {
    constructor(client) {
        this.client = client;
        this.pending = new Map(); // hid -> [{ type, key, original }]
    }

    /**
     * Apply optimistic update based on element data attributes.
     * Parses data-optimistic='{"class":"...","text":"...","attr":"...","value":"..."}' format
     * per spec Section 5.2.
     */
    applyOptimistic(el, eventType) {
        const hid = el.dataset.hid;
        if (!hid) return;

        // Parse JSON data-optimistic attribute
        const optimisticData = el.dataset.optimistic;
        if (!optimisticData) return;

        try {
            const config = JSON.parse(optimisticData);

            // Check for optimistic class toggle
            if (config.class) {
                this._applyClassOptimistic(el, hid, config.class);
            }

            // Check for optimistic text
            if (config.text) {
                this._applyTextOptimistic(el, hid, config.text);
            }

            // Check for optimistic attribute
            if (config.attr && config.value !== undefined) {
                this._applyAttrOptimistic(el, hid, config.attr, config.value);
            }
        } catch (e) {
            console.warn('[Vango] Invalid optimistic config:', e);
        }
    }

    /**
     * Apply optimistic class toggle
     * Format: "classname" or "classname:add" or "classname:remove" or "classname:toggle"
     */
    _applyClassOptimistic(el, hid, classConfig) {
        const [className, action = 'toggle'] = classConfig.split(':');

        // Store original state
        const original = el.classList.contains(className);

        // Apply change
        switch (action) {
            case 'add':
                el.classList.add(className);
                break;
            case 'remove':
                el.classList.remove(className);
                break;
            case 'toggle':
            default:
                el.classList.toggle(className);
        }

        // Track for potential revert
        this._trackPending(hid, 'class', className, original);
    }

    /**
     * Apply optimistic text change
     */
    _applyTextOptimistic(el, hid, text) {
        // Store original
        const original = el.textContent;

        // Apply change
        el.textContent = text;

        // Track for revert
        this._trackPending(hid, 'text', null, original);
    }

    /**
     * Apply optimistic attribute change
     */
    _applyAttrOptimistic(el, hid, attr, value) {
        // SECURITY: Block on* attributes to prevent XSS
        if (attr.length > 2 && attr.substring(0, 2).toLowerCase() === 'on') {
            console.warn('[Vango] Blocked dangerous optimistic attribute:', attr);
            return;
        }

        // Store original
        const original = el.getAttribute(attr);

        // Apply change
        if (value === 'null' || value === '') {
            el.removeAttribute(attr);
        } else {
            el.setAttribute(attr, value);
        }

        // Track for revert
        this._trackPending(hid, 'attr', attr, original);
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
            if (!el) continue;

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
            case 'class':
                if (update.original) {
                    el.classList.add(update.key);
                } else {
                    el.classList.remove(update.key);
                }
                break;
            case 'text':
                el.textContent = update.original;
                break;
            case 'attr':
                if (update.original === null) {
                    el.removeAttribute(update.key);
                } else {
                    el.setAttribute(update.key, update.original);
                }
                break;
        }
    }
}
