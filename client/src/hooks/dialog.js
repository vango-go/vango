/**
 * Dialog Hook - Modal Dialog Behavior
 *
 * Provides focus trapping, escape key handling, and click-outside closing
 * for modal dialogs. Integrates with the FocusTrap hook for accessibility.
 *
 * Config options:
 * - closeOnEscape: boolean (default: true) - Close on Escape key
 * - closeOnOutside: boolean (default: true) - Close on click outside
 * - initialFocus: string (optional) - CSS selector for initial focus element
 */

export class DialogHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        // Store previously focused element
        this.previousFocus = document.activeElement;

        // Setup event handlers
        this._onKeyDown = this._handleKeyDown.bind(this);
        this._onClick = this._handleClick.bind(this);

        document.addEventListener('keydown', this._onKeyDown);

        if (config.closeOnOutside !== false) {
            // Use setTimeout to avoid immediate trigger from the opening click
            setTimeout(() => {
                document.addEventListener('click', this._onClick);
            }, 0);
        }

        // Focus management
        this._focusFirst();

        // Prevent body scroll
        this._originalOverflow = document.body.style.overflow;
        document.body.style.overflow = 'hidden';
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
    }

    destroyed(el) {
        document.removeEventListener('keydown', this._onKeyDown);
        document.removeEventListener('click', this._onClick);

        // Restore body scroll
        document.body.style.overflow = this._originalOverflow || '';

        // Restore focus
        if (this.previousFocus && typeof this.previousFocus.focus === 'function') {
            this.previousFocus.focus();
        }
    }

    _handleKeyDown(e) {
        if (e.key === 'Escape' && this.config.closeOnEscape !== false) {
            e.preventDefault();
            this.pushEvent('close', {});
            return;
        }

        // Focus trapping
        if (e.key === 'Tab') {
            this._trapFocus(e);
        }
    }

    _handleClick(e) {
        // Check if click is outside the dialog content
        if (!this.el.contains(e.target)) {
            this.pushEvent('close', {});
        }
    }

    _focusFirst() {
        // Try initial focus element first
        if (this.config.initialFocus) {
            const initial = this.el.querySelector(this.config.initialFocus);
            if (initial) {
                initial.focus();
                return;
            }
        }

        // Find first focusable element
        const focusable = this._getFocusableElements();
        if (focusable.length > 0) {
            focusable[0].focus();
        } else {
            // Fallback: make container focusable
            this.el.setAttribute('tabindex', '-1');
            this.el.focus();
        }
    }

    _trapFocus(e) {
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
