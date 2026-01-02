/**
 * Collapsible Hook - Expand/Collapse Animation
 *
 * Provides smooth height animation for collapsible content.
 * Fires 'toggle' event when state changes.
 *
 * Config options:
 * - open: boolean (default: false) - Initial open state
 * - duration: number (default: 200) - Animation duration in ms
 */

export class CollapsibleHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        this.duration = config.duration || 200;
        this.isOpen = config.open !== undefined ? config.open : false;
        this.animating = false;

        // Store original styles
        this._originalOverflow = el.style.overflow;
        this._originalHeight = el.style.height;

        // Set initial state
        if (!this.isOpen) {
            this.el.style.height = '0';
            this.el.style.overflow = 'hidden';
        }

        // Find trigger element (sibling or parent button)
        this._findTrigger();
    }

    updated(el, config, pushEvent) {
        this.pushEvent = pushEvent;
        this.duration = config.duration || 200;

        // Handle external state changes
        const newOpen = config.open !== undefined ? config.open : this.isOpen;
        if (newOpen !== this.isOpen) {
            if (newOpen) {
                this._expand();
            } else {
                this._collapse();
            }
        }
    }

    destroyed(el) {
        if (this._trigger) {
            this._trigger.removeEventListener('click', this._onTriggerClick);
        }
    }

    _findTrigger() {
        // Look for trigger in previous sibling or parent
        const triggerSelector = this.config.trigger || '[data-collapsible-trigger]';

        // Check previous sibling
        let trigger = this.el.previousElementSibling;
        if (trigger && trigger.matches(triggerSelector)) {
            this._trigger = trigger;
        } else {
            // Check parent for trigger
            trigger = this.el.parentElement?.querySelector(triggerSelector);
            if (trigger && trigger !== this.el) {
                this._trigger = trigger;
            }
        }

        if (this._trigger) {
            this._onTriggerClick = this._handleTriggerClick.bind(this);
            this._trigger.addEventListener('click', this._onTriggerClick);
        }
    }

    _handleTriggerClick(e) {
        e.preventDefault();
        this.toggle();
    }

    toggle() {
        if (this.animating) return;

        if (this.isOpen) {
            this._collapse();
        } else {
            this._expand();
        }
    }

    _expand() {
        if (this.animating || this.isOpen) return;

        this.animating = true;
        this.isOpen = true;

        // Get target height
        this.el.style.height = 'auto';
        const targetHeight = this.el.scrollHeight;
        this.el.style.height = '0';
        this.el.style.overflow = 'hidden';

        // Force reflow
        this.el.offsetHeight;

        // Animate
        this.el.style.transition = `height ${this.duration}ms ease`;
        this.el.style.height = `${targetHeight}px`;

        setTimeout(() => {
            this.el.style.transition = '';
            this.el.style.height = 'auto';
            this.el.style.overflow = this._originalOverflow || '';
            this.animating = false;

            this.pushEvent('toggle', { open: true });
        }, this.duration);
    }

    _collapse() {
        if (this.animating || !this.isOpen) return;

        this.animating = true;
        this.isOpen = false;

        // Set current height explicitly
        const currentHeight = this.el.scrollHeight;
        this.el.style.height = `${currentHeight}px`;
        this.el.style.overflow = 'hidden';

        // Force reflow
        this.el.offsetHeight;

        // Animate
        this.el.style.transition = `height ${this.duration}ms ease`;
        this.el.style.height = '0';

        setTimeout(() => {
            this.el.style.transition = '';
            this.animating = false;

            this.pushEvent('toggle', { open: false });
        }, this.duration);
    }
}
