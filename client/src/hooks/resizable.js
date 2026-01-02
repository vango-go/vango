/**
 * Resizable Hook - Resize Handles
 *
 * Provides resize handles on elements.
 * Fires 'resize' event when resize completes.
 *
 * Config options:
 * - handles: string (default: 'se') - Comma-separated handles: n,e,s,w,ne,se,sw,nw
 * - minWidth: number (optional) - Minimum width in pixels
 * - maxWidth: number (optional) - Maximum width in pixels
 * - minHeight: number (optional) - Minimum height in pixels
 * - maxHeight: number (optional) - Maximum height in pixels
 */

export class ResizableHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        this.handles = (config.handles || 'se').split(',').map(h => h.trim());
        this.minWidth = config.minWidth || 0;
        this.maxWidth = config.maxWidth || Infinity;
        this.minHeight = config.minHeight || 0;
        this.maxHeight = config.maxHeight || Infinity;

        this.resizing = false;
        this.currentHandle = null;
        this.startX = 0;
        this.startY = 0;
        this.startWidth = 0;
        this.startHeight = 0;
        this.startLeft = 0;
        this.startTop = 0;

        this._createHandles();
        this._bindEvents();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
        this.minWidth = config.minWidth || 0;
        this.maxWidth = config.maxWidth || Infinity;
        this.minHeight = config.minHeight || 0;
        this.maxHeight = config.maxHeight || Infinity;
    }

    destroyed(el) {
        this._unbindEvents();
        this._removeHandles();
    }

    _createHandles() {
        // Ensure element is positioned for handles to work
        const style = getComputedStyle(this.el);
        if (style.position === 'static') {
            this.el.style.position = 'relative';
        }

        this._handleElements = [];

        const handleSize = 8;
        const halfSize = handleSize / 2;

        const positions = {
            n:  { top: `-${halfSize}px`, left: '50%', width: '50%', height: `${handleSize}px`, cursor: 'n-resize', transform: 'translateX(-50%)' },
            s:  { bottom: `-${halfSize}px`, left: '50%', width: '50%', height: `${handleSize}px`, cursor: 's-resize', transform: 'translateX(-50%)' },
            e:  { right: `-${halfSize}px`, top: '50%', width: `${handleSize}px`, height: '50%', cursor: 'e-resize', transform: 'translateY(-50%)' },
            w:  { left: `-${halfSize}px`, top: '50%', width: `${handleSize}px`, height: '50%', cursor: 'w-resize', transform: 'translateY(-50%)' },
            ne: { top: `-${halfSize}px`, right: `-${halfSize}px`, width: `${handleSize}px`, height: `${handleSize}px`, cursor: 'ne-resize' },
            se: { bottom: `-${halfSize}px`, right: `-${halfSize}px`, width: `${handleSize}px`, height: `${handleSize}px`, cursor: 'se-resize' },
            sw: { bottom: `-${halfSize}px`, left: `-${halfSize}px`, width: `${handleSize}px`, height: `${handleSize}px`, cursor: 'sw-resize' },
            nw: { top: `-${halfSize}px`, left: `-${halfSize}px`, width: `${handleSize}px`, height: `${handleSize}px`, cursor: 'nw-resize' },
        };

        for (const handle of this.handles) {
            if (!positions[handle]) continue;

            const handleEl = document.createElement('div');
            handleEl.className = `vango-resize-handle vango-resize-${handle}`;
            handleEl.dataset.handle = handle;

            const pos = positions[handle];
            handleEl.style.cssText = `
                position: absolute;
                background: transparent;
                z-index: 10;
                ${pos.top ? `top: ${pos.top};` : ''}
                ${pos.bottom ? `bottom: ${pos.bottom};` : ''}
                ${pos.left ? `left: ${pos.left};` : ''}
                ${pos.right ? `right: ${pos.right};` : ''}
                width: ${pos.width};
                height: ${pos.height};
                cursor: ${pos.cursor};
                ${pos.transform ? `transform: ${pos.transform};` : ''}
            `;

            this.el.appendChild(handleEl);
            this._handleElements.push(handleEl);
        }
    }

    _removeHandles() {
        for (const handle of this._handleElements) {
            handle.remove();
        }
        this._handleElements = [];
    }

    _bindEvents() {
        this._onMouseDown = this._handleMouseDown.bind(this);
        this._onMouseMove = this._handleMouseMove.bind(this);
        this._onMouseUp = this._handleMouseUp.bind(this);

        for (const handle of this._handleElements) {
            handle.addEventListener('mousedown', this._onMouseDown);
        }
        document.addEventListener('mousemove', this._onMouseMove);
        document.addEventListener('mouseup', this._onMouseUp);

        // Touch support
        this._onTouchStart = this._handleTouchStart.bind(this);
        this._onTouchMove = this._handleTouchMove.bind(this);
        this._onTouchEnd = this._handleTouchEnd.bind(this);

        for (const handle of this._handleElements) {
            handle.addEventListener('touchstart', this._onTouchStart, { passive: false });
        }
        document.addEventListener('touchmove', this._onTouchMove, { passive: false });
        document.addEventListener('touchend', this._onTouchEnd);
    }

    _unbindEvents() {
        for (const handle of this._handleElements) {
            handle.removeEventListener('mousedown', this._onMouseDown);
            handle.removeEventListener('touchstart', this._onTouchStart);
        }
        document.removeEventListener('mousemove', this._onMouseMove);
        document.removeEventListener('mouseup', this._onMouseUp);
        document.removeEventListener('touchmove', this._onTouchMove);
        document.removeEventListener('touchend', this._onTouchEnd);
    }

    _handleMouseDown(e) {
        e.preventDefault();
        this._startResize(e.target.dataset.handle, e.clientX, e.clientY);
    }

    _handleTouchStart(e) {
        e.preventDefault();
        const touch = e.touches[0];
        this._startResize(e.target.dataset.handle, touch.clientX, touch.clientY);
    }

    _startResize(handle, x, y) {
        this.resizing = true;
        this.currentHandle = handle;
        this.startX = x;
        this.startY = y;

        const rect = this.el.getBoundingClientRect();
        this.startWidth = rect.width;
        this.startHeight = rect.height;
        this.startLeft = rect.left;
        this.startTop = rect.top;

        this.el.classList.add('resizing');
    }

    _handleMouseMove(e) {
        if (!this.resizing) return;
        e.preventDefault();
        this._updateSize(e.clientX, e.clientY);
    }

    _handleTouchMove(e) {
        if (!this.resizing) return;
        e.preventDefault();
        const touch = e.touches[0];
        this._updateSize(touch.clientX, touch.clientY);
    }

    _updateSize(x, y) {
        const deltaX = x - this.startX;
        const deltaY = y - this.startY;

        let newWidth = this.startWidth;
        let newHeight = this.startHeight;

        const handle = this.currentHandle;

        // Handle east (right) edge
        if (handle.includes('e')) {
            newWidth = this.startWidth + deltaX;
        }
        // Handle west (left) edge
        if (handle.includes('w')) {
            newWidth = this.startWidth - deltaX;
        }
        // Handle south (bottom) edge
        if (handle.includes('s')) {
            newHeight = this.startHeight + deltaY;
        }
        // Handle north (top) edge
        if (handle.includes('n')) {
            newHeight = this.startHeight - deltaY;
        }

        // Apply constraints
        newWidth = Math.max(this.minWidth, Math.min(this.maxWidth, newWidth));
        newHeight = Math.max(this.minHeight, Math.min(this.maxHeight, newHeight));

        // Apply new dimensions
        this.el.style.width = `${newWidth}px`;
        this.el.style.height = `${newHeight}px`;
    }

    _handleMouseUp(e) {
        if (!this.resizing) return;
        this._endResize();
    }

    _handleTouchEnd(e) {
        if (!this.resizing) return;
        this._endResize();
    }

    _endResize() {
        this.resizing = false;
        this.el.classList.remove('resizing');

        const rect = this.el.getBoundingClientRect();

        this.pushEvent('resize', {
            width: Math.round(rect.width),
            height: Math.round(rect.height),
        });

        this.currentHandle = null;
    }
}
