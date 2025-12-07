/**
 * FocusTrap Hook - Modal Accessibility
 *
 * Constrains keyboard navigation (Tab/Shift+Tab) to a specific container.
 * Critical for Modal/Dialog accessibility compliance (WCAG 2.1).
 *
 * VangoUI Integration: Used internally by ui.Dialog and ui.Modal components.
 */

export class FocusTrapHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.pushEvent = pushEvent;
        this.active = config.active !== false;

        // Save previously focused element to restore later
        this.restoreFocusTo = document.activeElement;

        this._onKeyDown = this._handleKeyDown.bind(this);
        this.el.addEventListener('keydown', this._onKeyDown);

        if (this.active) {
            this._focusFirst();
        }
    }

    updated(el, config, pushEvent) {
        this.active = config.active !== false;
        this.pushEvent = pushEvent;
    }

    destroyed(el) {
        this.el.removeEventListener('keydown', this._onKeyDown);

        // Restore focus on close
        if (this.restoreFocusTo && typeof this.restoreFocusTo.focus === 'function') {
            this.restoreFocusTo.focus();
        }
    }

    _handleKeyDown(e) {
        if (!this.active || e.key !== 'Tab') return;

        const focusable = this._getFocusableElements();

        if (focusable.length === 0) {
            e.preventDefault();
            return;
        }

        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (e.shiftKey) {
            // Shift + Tab
            if (document.activeElement === first) {
                e.preventDefault();
                last.focus();
            }
        } else {
            // Tab
            if (document.activeElement === last) {
                e.preventDefault();
                first.focus();
            }
        }
    }

    _focusFirst() {
        const focusable = this._getFocusableElements();
        if (focusable.length > 0) {
            focusable[0].focus();
        } else {
            // Fallback: focus container itself with tabindex
            this.el.setAttribute('tabindex', '-1');
            this.el.focus();
        }
    }

    _getFocusableElements() {
        return this.el.querySelectorAll(
            'button:not([disabled]), ' +
            '[href], ' +
            'input:not([disabled]), ' +
            'select:not([disabled]), ' +
            'textarea:not([disabled]), ' +
            '[tabindex]:not([tabindex="-1"])'
        );
    }
}
