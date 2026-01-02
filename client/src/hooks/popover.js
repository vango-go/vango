/**
 * Popover Hook - Floating Content Positioning
 *
 * Handles positioning and click-outside closing for popovers.
 * Automatically repositions when viewport edge is reached.
 *
 * Config options:
 * - closeOnEscape: boolean (default: true) - Close on Escape key
 * - closeOnOutside: boolean (default: true) - Close on click outside
 * - side: string (default: 'bottom') - Preferred side (top, right, bottom, left)
 * - align: string (default: 'center') - Alignment (start, center, end)
 * - offset: number (default: 4) - Gap between trigger and content in pixels
 */

export class PopoverHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        // Find trigger and content elements
        this.trigger = el.querySelector('[data-popover-trigger]');
        this.content = el.querySelector('[data-popover-content]');

        if (!this.content) {
            // If no explicit content marker, assume the hook element is the content
            this.content = el;
        }

        // Setup event handlers
        this._onKeyDown = this._handleKeyDown.bind(this);
        this._onClick = this._handleClick.bind(this);
        this._onScroll = this._handleScroll.bind(this);
        this._onResize = this._handleResize.bind(this);

        document.addEventListener('keydown', this._onKeyDown);

        if (config.closeOnOutside !== false) {
            setTimeout(() => {
                document.addEventListener('click', this._onClick);
            }, 0);
        }

        // Position tracking
        window.addEventListener('scroll', this._onScroll, true);
        window.addEventListener('resize', this._onResize);

        // Initial positioning
        this._position();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
        this._position();
    }

    destroyed(el) {
        document.removeEventListener('keydown', this._onKeyDown);
        document.removeEventListener('click', this._onClick);
        window.removeEventListener('scroll', this._onScroll, true);
        window.removeEventListener('resize', this._onResize);
    }

    _handleKeyDown(e) {
        if (e.key === 'Escape' && this.config.closeOnEscape !== false) {
            e.preventDefault();
            this.pushEvent('close', {});
        }
    }

    _handleClick(e) {
        if (!this.el.contains(e.target)) {
            this.pushEvent('close', {});
        }
    }

    _handleScroll() {
        this._position();
    }

    _handleResize() {
        this._position();
    }

    _position() {
        if (!this.trigger || !this.content) return;

        const triggerRect = this.trigger.getBoundingClientRect();
        const contentRect = this.content.getBoundingClientRect();
        const offset = this.config.offset || 4;

        let side = this.config.side || 'bottom';
        let align = this.config.align || 'center';

        // Calculate position based on side
        let top, left;

        switch (side) {
            case 'top':
                top = triggerRect.top - contentRect.height - offset;
                // Flip if would overflow
                if (top < 0) {
                    top = triggerRect.bottom + offset;
                    side = 'bottom';
                }
                break;
            case 'bottom':
                top = triggerRect.bottom + offset;
                // Flip if would overflow
                if (top + contentRect.height > window.innerHeight) {
                    top = triggerRect.top - contentRect.height - offset;
                    side = 'top';
                }
                break;
            case 'left':
                left = triggerRect.left - contentRect.width - offset;
                // Flip if would overflow
                if (left < 0) {
                    left = triggerRect.right + offset;
                    side = 'right';
                }
                break;
            case 'right':
                left = triggerRect.right + offset;
                // Flip if would overflow
                if (left + contentRect.width > window.innerWidth) {
                    left = triggerRect.left - contentRect.width - offset;
                    side = 'left';
                }
                break;
        }

        // Calculate alignment for horizontal sides (top/bottom)
        if (side === 'top' || side === 'bottom') {
            switch (align) {
                case 'start':
                    left = triggerRect.left;
                    break;
                case 'center':
                    left = triggerRect.left + (triggerRect.width - contentRect.width) / 2;
                    break;
                case 'end':
                    left = triggerRect.right - contentRect.width;
                    break;
            }

            // Ensure doesn't overflow horizontally
            if (left < 0) left = 0;
            if (left + contentRect.width > window.innerWidth) {
                left = window.innerWidth - contentRect.width;
            }
        }

        // Calculate alignment for vertical sides (left/right)
        if (side === 'left' || side === 'right') {
            switch (align) {
                case 'start':
                    top = triggerRect.top;
                    break;
                case 'center':
                    top = triggerRect.top + (triggerRect.height - contentRect.height) / 2;
                    break;
                case 'end':
                    top = triggerRect.bottom - contentRect.height;
                    break;
            }

            // Ensure doesn't overflow vertically
            if (top < 0) top = 0;
            if (top + contentRect.height > window.innerHeight) {
                top = window.innerHeight - contentRect.height;
            }
        }

        // Apply position
        this.content.style.position = 'fixed';
        this.content.style.top = `${top}px`;
        this.content.style.left = `${left}px`;

        // Update data-side attribute for styling
        this.content.dataset.side = side;
    }
}
