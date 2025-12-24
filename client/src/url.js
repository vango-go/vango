/**
 * URL Manager
 *
 * Handles URL query parameter updates from server patches.
 * Supports push/replace history modes.
 */

/**
 * URLManager handles URL-related patches from the server.
 */
export class URLManager {
    constructor(client, options = {}) {
        this.client = client;
        this.options = {
            debug: options.debug || false,
            ...options,
        };

        // Pending updates for debouncing (not used client-side, but structure matches server)
        this.pending = new Map();
    }

    /**
     * Apply a URL patch
     * @param {Object} patch - The URL patch with op and params
     */
    applyPatch(patch) {
        const { op, params } = patch;

        if (!params || typeof params !== 'object') {
            if (this.options.debug) {
                console.warn('[Vango URL] Invalid params:', params);
            }
            return;
        }

        // Build new URL
        const url = new URL(window.location);

        // Apply param changes
        for (const [key, value] of Object.entries(params)) {
            if (value === '' || value === null || value === undefined) {
                url.searchParams.delete(key);
            } else {
                url.searchParams.set(key, value);
            }
        }

        // Determine mode from op
        const isPush = op === 0x30; // PatchURLPush

        if (this.options.debug) {
            console.log('[Vango URL]', isPush ? 'Push' : 'Replace', url.toString());
        }

        // Update history
        if (isPush) {
            history.pushState(null, '', url.toString());
        } else {
            history.replaceState(null, '', url.toString());
        }

        // Dispatch custom event for app to listen
        this._dispatchEvent(params, isPush);
    }

    /**
     * Dispatch custom event for URL changes
     */
    _dispatchEvent(params, isPush) {
        const event = new CustomEvent('vango:url', {
            detail: {
                params,
                mode: isPush ? 'push' : 'replace',
                url: window.location.href,
            },
            bubbles: true,
        });
        document.dispatchEvent(event);
    }

    /**
     * Get current query parameters as an object
     */
    getParams() {
        const params = {};
        const searchParams = new URLSearchParams(window.location.search);
        for (const [key, value] of searchParams) {
            params[key] = value;
        }
        return params;
    }

    /**
     * Get a specific query parameter
     */
    getParam(key) {
        const searchParams = new URLSearchParams(window.location.search);
        return searchParams.get(key);
    }

    /**
     * Check if a query parameter exists
     */
    hasParam(key) {
        const searchParams = new URLSearchParams(window.location.search);
        return searchParams.has(key);
    }
}

/**
 * Default export for convenience
 */
export default URLManager;
