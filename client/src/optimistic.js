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
     * Apply optimistic update based on element data attributes
     */
    applyOptimistic(el, eventType) {
        const hid = el.dataset.hid;
        if (!hid) return;

        // Check for optimistic class toggle
        const optimisticClass = el.dataset.optimisticClass;
        if (optimisticClass) {
            this._applyClassOptimistic(el, hid, optimisticClass);
        }

        // Check for optimistic text
        const optimisticText = el.dataset.optimisticText;
        if (optimisticText) {
            this._applyTextOptimistic(el, hid, optimisticText);
        }

        // Check for optimistic attribute
        const optimisticAttr = el.dataset.optimisticAttr;
        const optimisticValue = el.dataset.optimisticValue;
        if (optimisticAttr && optimisticValue !== undefined) {
            this._applyAttrOptimistic(el, hid, optimisticAttr, optimisticValue);
        }

        // Check for parent optimistic class
        const parentOptimisticClass = el.dataset.optimisticParentClass;
        if (parentOptimisticClass && el.parentElement) {
            const parentHid = el.parentElement.dataset.hid || `parent-${hid}`;
            this._applyClassOptimistic(el.parentElement, parentHid, parentOptimisticClass);
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
