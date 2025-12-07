/**
 * Tooltip Hook
 *
 * Simple tooltip display on hover.
 */

export class TooltipHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.tooltip = null;

        this.content = config.content || '';
        this.placement = config.placement || 'top';
        this.delay = config.delay || 200;

        this._bindEvents();
    }

    updated(el, config, pushEvent) {
        this.content = config.content || '';
        this.placement = config.placement || 'top';

        if (this.tooltip) {
            this.tooltip.textContent = this.content;
        }
    }

    destroyed(el) {
        this._unbindEvents();
        this._hideTooltip();
    }

    _bindEvents() {
        this._onMouseEnter = this._handleMouseEnter.bind(this);
        this._onMouseLeave = this._handleMouseLeave.bind(this);

        this.el.addEventListener('mouseenter', this._onMouseEnter);
        this.el.addEventListener('mouseleave', this._onMouseLeave);
    }

    _unbindEvents() {
        this.el.removeEventListener('mouseenter', this._onMouseEnter);
        this.el.removeEventListener('mouseleave', this._onMouseLeave);
    }

    _handleMouseEnter() {
        this._showTimer = setTimeout(() => {
            this._showTooltip();
        }, this.delay);
    }

    _handleMouseLeave() {
        clearTimeout(this._showTimer);
        this._hideTooltip();
    }

    _showTooltip() {
        if (!this.content) return;

        // Create tooltip element
        this.tooltip = document.createElement('div');
        this.tooltip.className = 'vango-tooltip';
        this.tooltip.textContent = this.content;
        this.tooltip.style.cssText = `
            position: fixed;
            z-index: 10000;
            padding: 4px 8px;
            background: #333;
            color: white;
            border-radius: 4px;
            font-size: 12px;
            pointer-events: none;
            white-space: nowrap;
        `;

        document.body.appendChild(this.tooltip);

        // Position after adding to DOM (so we can get dimensions)
        this._position();
    }

    _position() {
        if (!this.tooltip) return;

        const rect = this.el.getBoundingClientRect();
        const tipRect = this.tooltip.getBoundingClientRect();

        let top, left;

        switch (this.placement) {
            case 'top':
                top = rect.top - tipRect.height - 8;
                left = rect.left + (rect.width - tipRect.width) / 2;
                break;
            case 'bottom':
                top = rect.bottom + 8;
                left = rect.left + (rect.width - tipRect.width) / 2;
                break;
            case 'left':
                top = rect.top + (rect.height - tipRect.height) / 2;
                left = rect.left - tipRect.width - 8;
                break;
            case 'right':
                top = rect.top + (rect.height - tipRect.height) / 2;
                left = rect.right + 8;
                break;
            default:
                top = rect.top - tipRect.height - 8;
                left = rect.left + (rect.width - tipRect.width) / 2;
        }

        this.tooltip.style.top = `${top}px`;
        this.tooltip.style.left = `${left}px`;
    }

    _hideTooltip() {
        if (this.tooltip) {
            this.tooltip.remove();
            this.tooltip = null;
        }
    }
}
