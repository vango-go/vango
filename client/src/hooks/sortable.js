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
        this._startDrag(item, e.clientX, e.clientY);
    }

    _handleTouchStart(e) {
        const item = this._findItem(e.target);
        if (!item) return;

        if (this.handle && !e.target.closest(this.handle)) return;

        e.preventDefault();
        const touch = e.touches[0];
        this._startDrag(item, touch.clientX, touch.clientY);
    }

    _findItem(target) {
        // Find direct child of container
        let item = target;
        while (item && item.parentElement !== this.el) {
            item = item.parentElement;
        }
        return item;
    }

    _startDrag(item, x, y) {
        this.dragging = item;
        this.activeContainer = item.parentElement;
        this.initialContainer = this.activeContainer;
        this.startIndex = Array.from(this.activeContainer.children).indexOf(item);
        this.startY = y;
        this.startX = x;
        this.itemHeight = item.offsetHeight;

        // Store initial positions
        const rect = item.getBoundingClientRect();
        this.ghostStartTop = rect.top;
        this.ghostStartLeft = rect.left;

        // Create ghost
        this.ghost = item.cloneNode(true);
        this.ghost.classList.add(this.ghostClass);
        this.ghost.style.position = 'fixed';
        this.ghost.style.zIndex = '9999';
        this.ghost.style.width = `${item.offsetWidth}px`;
        this.ghost.style.height = `${item.offsetHeight}px`;
        this.ghost.style.left = `${this.ghostStartLeft}px`;
        this.ghost.style.top = `${this.ghostStartTop}px`;
        this.ghost.style.pointerEvents = 'none';
        this.ghost.style.opacity = '0.8';
        this.ghost.style.transition = 'none';
        document.body.appendChild(this.ghost);

        // Style dragging item
        item.classList.add(this.dragClass);
        item.style.opacity = '0.4';
    }

    _handleMouseMove(e) {
        if (!this.dragging) return;
        e.preventDefault();
        this._updateDrag(e.clientX, e.clientY);
    }

    _handleTouchMove(e) {
        if (!this.dragging) return;
        e.preventDefault();
        const touch = e.touches[0];
        this._updateDrag(touch.clientX, touch.clientY);
    }

    _updateDrag(x, y) {
        // Move ghost
        const deltaY = y - this.startY;
        const deltaX = x - this.startX;
        if (this.ghost) {
            this.ghost.style.top = `${this.ghostStartTop + deltaY}px`;
            this.ghost.style.left = `${this.ghostStartLeft + deltaX}px`;
        }

        // Check for cross-container drag
        // We peek at the element under cursor to see if we moved to another sortable list
        // Note: we must temporarily hide ghost to not hit it with elementFromPoint
        this.ghost.style.display = 'none';
        const elUnderCursor = document.elementFromPoint(x, y);
        this.ghost.style.display = '';

        if (elUnderCursor) {
            const targetContainer = elUnderCursor.closest('[data-hook="Sortable"]');

            // If we found a container, it's different from current, and shares the same group
            if (targetContainer &&
                targetContainer !== this.activeContainer &&
                targetContainer.dataset.hookConfig) {

                try {
                    const targetConfig = JSON.parse(targetContainer.dataset.hookConfig);
                    if (targetConfig.group === this.config.group) {
                        // Move dragging item to new container
                        this.activeContainer = targetContainer;
                        this.activeContainer.appendChild(this.dragging);
                        // Config update will happen on next render or we can assume same behavior
                    }
                } catch (e) {
                    // Ignore invalid config
                }
            }
        }

        // Find insert position within current container (this.activeContainer)
        const children = Array.from(this.activeContainer.children);
        const currentIndex = children.indexOf(this.dragging);

        for (let i = 0; i < children.length; i++) {
            if (i === currentIndex) continue;

            const child = children[i];
            const rect = child.getBoundingClientRect();

            // Simple logic: if overlapping significantly or past midpoint
            // Vertical list logic
            const midpoint = rect.top + rect.height / 2;

            if (y < midpoint && i < currentIndex) {
                this.activeContainer.insertBefore(this.dragging, child);
                break;
            } else if (y > midpoint && i > currentIndex) {
                this.activeContainer.insertBefore(this.dragging, child.nextSibling);
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
        const endIndex = Array.from(this.activeContainer.children).indexOf(this.dragging);
        const id = this.dragging.dataset.id || this.dragging.dataset.hid;
        const targetContainerHid = this.activeContainer.dataset.hid;

        // Cleanup
        this.dragging.classList.remove(this.dragClass);
        this.dragging.style.opacity = '';
        if (this.ghost) {
            this.ghost.remove();
        }

        // Send event if position changed OR container changed
        if (this.activeContainer !== this.initialContainer || endIndex !== this.startIndex) {
            this.pushEvent('reorder', {
                id: id,
                fromContainer: this.initialContainer.dataset.id || this.initialContainer.dataset.hid,
                toContainer: this.activeContainer.dataset.id || this.activeContainer.dataset.hid,
                fromIndex: this.startIndex,
                toIndex: endIndex,
            });
        }

        this.dragging = null;
        this.ghost = null;
        this.activeContainer = null;
        this.initialContainer = null;
    }
}
