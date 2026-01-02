/**
 * Collapsible Hook - Expand/Collapse Animation
 *
 * Provides smooth height animation. Fires 'toggle' event when state changes.
 * Config: open (default: false), duration (default: 200ms)
 */

export class CollapsibleHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;
        this.duration = config.duration || 200;
        this.isOpen = !!config.open;
        this.animating = false;

        if (!this.isOpen) {
            el.style.height = '0';
            el.style.overflow = 'hidden';
        }
    }

    updated(el, config, pushEvent) {
        this.pushEvent = pushEvent;
        this.duration = config.duration || 200;
        const newOpen = !!config.open;
        if (newOpen !== this.isOpen) {
            newOpen ? this._expand() : this._collapse();
        }
    }

    destroyed() {}

    _expand() {
        if (this.animating || this.isOpen) return;
        this.animating = true;
        this.isOpen = true;

        const el = this.el;
        el.style.height = 'auto';
        const h = el.scrollHeight;
        el.style.height = '0';
        el.style.overflow = 'hidden';
        el.offsetHeight; // reflow
        el.style.transition = `height ${this.duration}ms ease`;
        el.style.height = h + 'px';

        setTimeout(() => {
            el.style.transition = '';
            el.style.height = 'auto';
            el.style.overflow = '';
            this.animating = false;
            this.pushEvent('toggle', { open: true });
        }, this.duration);
    }

    _collapse() {
        if (this.animating || !this.isOpen) return;
        this.animating = true;
        this.isOpen = false;

        const el = this.el;
        el.style.height = el.scrollHeight + 'px';
        el.style.overflow = 'hidden';
        el.offsetHeight; // reflow
        el.style.transition = `height ${this.duration}ms ease`;
        el.style.height = '0';

        setTimeout(() => {
            el.style.transition = '';
            this.animating = false;
            this.pushEvent('toggle', { open: false });
        }, this.duration);
    }
}
