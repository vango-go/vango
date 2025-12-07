/**
 * Dropdown Hook
 *
 * Provides click-outside-to-close behavior for dropdowns/modals.
 */

export class DropdownHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        this.closeOnEscape = config.closeOnEscape !== false;
        this.closeOnClickOutside = config.closeOnClickOutside !== false;

        this._bindEvents();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
    }

    destroyed(el) {
        this._unbindEvents();
    }

    _bindEvents() {
        this._onClickOutside = this._handleClickOutside.bind(this);
        this._onKeyDown = this._handleKeyDown.bind(this);

        // Delay to avoid immediate trigger from the click that opened the dropdown
        setTimeout(() => {
            document.addEventListener('click', this._onClickOutside);
            document.addEventListener('keydown', this._onKeyDown);
        }, 0);
    }

    _unbindEvents() {
        document.removeEventListener('click', this._onClickOutside);
        document.removeEventListener('keydown', this._onKeyDown);
    }

    _handleClickOutside(e) {
        if (!this.closeOnClickOutside) return;

        if (!this.el.contains(e.target)) {
            this.pushEvent('close', {});
        }
    }

    _handleKeyDown(e) {
        if (!this.closeOnEscape) return;

        if (e.key === 'Escape') {
            e.preventDefault();
            this.pushEvent('close', {});
        }
    }
}
