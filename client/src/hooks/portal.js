/**
 * Portal Hook - DOM Element Repositioning
 *
 * Moves the DOM element to the end of document.body to escape
 * overflow: hidden containers and ensure correct z-index stacking context.
 *
 * VangoUI Integration: Used internally by ui.Dialog, ui.Tooltip, ui.Popover.
 */

export class PortalHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.pushEvent = pushEvent;
        this.target = config.target || 'body';

        // Create placeholder to keep VNode tree structure aligned
        this.placeholder = document.createComment('portal-placeholder');

        // Insert placeholder before moving (only if has parent)
        if (el.parentNode) {
            el.parentNode.insertBefore(this.placeholder, el);
        }

        // Get or create portal container
        let portalRoot = document.getElementById('vango-portal-root');
        if (!portalRoot) {
            portalRoot = document.createElement('div');
            portalRoot.id = 'vango-portal-root';
            portalRoot.style.cssText = 'position: relative; z-index: 9999;';
            document.body.appendChild(portalRoot);
        }

        // Move element to portal
        portalRoot.appendChild(el);
    }

    updated(el, config, pushEvent) {
        this.pushEvent = pushEvent;
    }

    destroyed(el) {
        // Move back before destruction (optional, but good for cleanup)
        if (this.placeholder && this.placeholder.parentNode) {
            this.placeholder.parentNode.insertBefore(this.el, this.placeholder);
            this.placeholder.remove();
        }
    }
}

/**
 * Create portal root element on DOM ready.
 * Called from client init to ensure portal root exists early.
 */
export function ensurePortalRoot() {
    if (typeof document === 'undefined') return;

    let portalRoot = document.getElementById('vango-portal-root');
    if (!portalRoot) {
        portalRoot = document.createElement('div');
        portalRoot.id = 'vango-portal-root';
        portalRoot.style.cssText = 'position: relative; z-index: 9999;';
        document.body.appendChild(portalRoot);
    }
    return portalRoot;
}
