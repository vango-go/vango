/**
 * Sortable List Hook
 *
 * Provides drag-to-reorder functionality for lists.
 * Minimal implementation (~1KB vs 8KB for SortableJS).
 */

export class SortableHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        this.animation = config.animation || 150;
        this.handle = config.handle || null;
        this.ghostClass = config.ghostClass || 'sortable-ghost';
        this.dragClass = config.dragClass || 'sortable-drag';

        this.dragging = null;
        this.ghost = null;
        this.startIndex = -1;

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
        this._onMouseDown = this._handleMouseDown.bind(this);
        this._onMouseMove = this._handleMouseMove.bind(this);
        this._onMouseUp = this._handleMouseUp.bind(this);

        this.el.addEventListener('mousedown', this._onMouseDown);
        document.addEventListener('mousemove', this._onMouseMove);
        document.addEventListener('mouseup', this._onMouseUp);

        // Touch events
        this._onTouchStart = this._handleTouchStart.bind(this);
        this._onTouchMove = this._handleTouchMove.bind(this);
        this._onTouchEnd = this._handleTouchEnd.bind(this);

        this.el.addEventListener('touchstart', this._onTouchStart, { passive: false });
        document.addEventListener('touchmove', this._onTouchMove, { passive: false });
        document.addEventListener('touchend', this._onTouchEnd);
    }

    _unbindEvents() {
        this.el.removeEventListener('mousedown', this._onMouseDown);
        document.removeEventListener('mousemove', this._onMouseMove);
        document.removeEventListener('mouseup', this._onMouseUp);

        this.el.removeEventListener('touchstart', this._onTouchStart);
        document.removeEventListener('touchmove', this._onTouchMove);
        document.removeEventListener('touchend', this._onTouchEnd);
    }

    _handleMouseDown(e) {
        const item = this._findItem(e.target);
        if (!item) return;

        // Check handle
        if (this.handle && !e.target.closest(this.handle)) return;

        e.preventDefault();
        this._startDrag(item, e.clientY);
    }

    _handleTouchStart(e) {
        const item = this._findItem(e.target);
        if (!item) return;

        if (this.handle && !e.target.closest(this.handle)) return;

        e.preventDefault();
        const touch = e.touches[0];
        this._startDrag(item, touch.clientY);
    }

    _findItem(target) {
        // Find direct child of container
        let item = target;
        while (item && item.parentElement !== this.el) {
            item = item.parentElement;
        }
        return item;
    }

    _startDrag(item, y) {
        this.dragging = item;
        this.startIndex = Array.from(this.el.children).indexOf(item);
        this.startY = y;
        this.itemHeight = item.offsetHeight;

        // Create ghost
        this.ghost = item.cloneNode(true);
        this.ghost.classList.add(this.ghostClass);
        this.ghost.style.position = 'fixed';
        this.ghost.style.zIndex = '9999';
        this.ghost.style.width = `${item.offsetWidth}px`;
        this.ghost.style.left = `${item.getBoundingClientRect().left}px`;
        this.ghost.style.top = `${item.getBoundingClientRect().top}px`;
        this.ghost.style.pointerEvents = 'none';
        this.ghost.style.opacity = '0.8';
        document.body.appendChild(this.ghost);

        // Style dragging item
        item.classList.add(this.dragClass);
        item.style.opacity = '0.4';
    }

    _handleMouseMove(e) {
        if (!this.dragging) return;
        e.preventDefault();
        this._updateDrag(e.clientY);
    }

    _handleTouchMove(e) {
        if (!this.dragging) return;
        e.preventDefault();
        const touch = e.touches[0];
        this._updateDrag(touch.clientY);
    }

    _updateDrag(y) {
        // Move ghost
        const deltaY = y - this.startY;
        const startTop = this.el.children[this.startIndex]?.getBoundingClientRect().top;
        if (startTop !== undefined && this.ghost) {
            this.ghost.style.top = `${startTop + deltaY}px`;
        }

        // Find insert position
        const children = Array.from(this.el.children);
        const currentIndex = children.indexOf(this.dragging);

        for (let i = 0; i < children.length; i++) {
            if (i === currentIndex) continue;

            const child = children[i];
            const rect = child.getBoundingClientRect();
            const midpoint = rect.top + rect.height / 2;

            if (y < midpoint && i < currentIndex) {
                this.el.insertBefore(this.dragging, child);
                break;
            } else if (y > midpoint && i > currentIndex) {
                this.el.insertBefore(this.dragging, child.nextSibling);
                break;
            }
        }
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
        const endIndex = Array.from(this.el.children).indexOf(this.dragging);

        // Cleanup
        this.dragging.classList.remove(this.dragClass);
        this.dragging.style.opacity = '';
        if (this.ghost) {
            this.ghost.remove();
        }

        // Only send event if position changed
        if (endIndex !== this.startIndex) {
            const id = this.dragging.dataset.id || this.dragging.dataset.hid;

            this.pushEvent('reorder', {
                id: id,
                fromIndex: this.startIndex,
                toIndex: endIndex,
            });
        }

        this.dragging = null;
        this.ghost = null;
    }
}
