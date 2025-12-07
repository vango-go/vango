/**
 * Draggable Hook
 *
 * Provides free-form drag functionality for elements.
 */

export class DraggableHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        this.axis = config.axis || 'both'; // 'x', 'y', or 'both'
        this.handle = config.handle || null;
        this.bounds = config.bounds || null; // 'parent', 'window', or null

        this.dragging = false;
        this.startX = 0;
        this.startY = 0;
        this.currentX = 0;
        this.currentY = 0;

        this._bindEvents();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
        this.axis = config.axis || 'both';
        this.bounds = config.bounds || null;
    }

    destroyed(el) {
        this._unbindEvents();
    }

    _bindEvents() {
        this._onMouseDown = this._handleMouseDown.bind(this);
        this._onMouseMove = this._handleMouseMove.bind(this);
        this._onMouseUp = this._handleMouseUp.bind(this);

        const handleEl = this.handle ? this.el.querySelector(this.handle) : this.el;
        if (handleEl) {
            handleEl.addEventListener('mousedown', this._onMouseDown);
        }
        document.addEventListener('mousemove', this._onMouseMove);
        document.addEventListener('mouseup', this._onMouseUp);

        // Touch events
        this._onTouchStart = this._handleTouchStart.bind(this);
        this._onTouchMove = this._handleTouchMove.bind(this);
        this._onTouchEnd = this._handleTouchEnd.bind(this);

        if (handleEl) {
            handleEl.addEventListener('touchstart', this._onTouchStart, { passive: false });
        }
        document.addEventListener('touchmove', this._onTouchMove, { passive: false });
        document.addEventListener('touchend', this._onTouchEnd);
    }

    _unbindEvents() {
        const handleEl = this.handle ? this.el.querySelector(this.handle) : this.el;
        if (handleEl) {
            handleEl.removeEventListener('mousedown', this._onMouseDown);
            handleEl.removeEventListener('touchstart', this._onTouchStart);
        }
        document.removeEventListener('mousemove', this._onMouseMove);
        document.removeEventListener('mouseup', this._onMouseUp);
        document.removeEventListener('touchmove', this._onTouchMove);
        document.removeEventListener('touchend', this._onTouchEnd);
    }

    _handleMouseDown(e) {
        e.preventDefault();
        this._startDrag(e.clientX, e.clientY);
    }

    _handleTouchStart(e) {
        e.preventDefault();
        const touch = e.touches[0];
        this._startDrag(touch.clientX, touch.clientY);
    }

    _startDrag(x, y) {
        this.dragging = true;
        this.startX = x - this.currentX;
        this.startY = y - this.currentY;

        this.el.classList.add('dragging');
    }

    _handleMouseMove(e) {
        if (!this.dragging) return;
        e.preventDefault();
        this._updatePosition(e.clientX, e.clientY);
    }

    _handleTouchMove(e) {
        if (!this.dragging) return;
        e.preventDefault();
        const touch = e.touches[0];
        this._updatePosition(touch.clientX, touch.clientY);
    }

    _updatePosition(x, y) {
        let newX = x - this.startX;
        let newY = y - this.startY;

        // Apply axis constraint
        if (this.axis === 'x') {
            newY = this.currentY;
        } else if (this.axis === 'y') {
            newX = this.currentX;
        }

        // Apply bounds
        if (this.bounds) {
            const bounds = this._getBounds();
            newX = Math.max(bounds.minX, Math.min(bounds.maxX, newX));
            newY = Math.max(bounds.minY, Math.min(bounds.maxY, newY));
        }

        this.currentX = newX;
        this.currentY = newY;

        // Apply transform
        this.el.style.transform = `translate(${newX}px, ${newY}px)`;
    }

    _getBounds() {
        const rect = this.el.getBoundingClientRect();
        let container;

        if (this.bounds === 'parent') {
            container = this.el.parentElement.getBoundingClientRect();
        } else {
            container = {
                left: 0,
                top: 0,
                right: window.innerWidth,
                bottom: window.innerHeight,
            };
        }

        return {
            minX: container.left - rect.left + this.currentX,
            maxX: container.right - rect.right + this.currentX,
            minY: container.top - rect.top + this.currentY,
            maxY: container.bottom - rect.bottom + this.currentY,
        };
    }

    _handleMouseUp(e) {
        if (!this.dragging) return;
        this._endDrag();
    }

    _handleTouchEnd(e) {
        if (!this.dragging) return;
        this._endDrag();
    }

    _endDrag() {
        this.dragging = false;
        this.el.classList.remove('dragging');

        // Send position to server
        this.pushEvent('position', {
            x: this.currentX,
            y: this.currentY,
        });
    }
}
