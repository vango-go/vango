/**
 * Droppable Hook - Drop Zones
 *
 * Provides drop zone functionality for draggable elements.
 * Fires 'drop' event when a compatible element is dropped.
 *
 * Config options:
 * - accept: string (optional) - CSS selector for accepted draggables
 * - hoverClass: string (default: 'drag-over') - Class added when dragging over
 */

export class DroppableHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;

        this.accept = config.accept || null;
        this.hoverClass = config.hoverClass || 'drag-over';

        this._bindEvents();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
        this.accept = config.accept || null;
        this.hoverClass = config.hoverClass || 'drag-over';
    }

    destroyed(el) {
        this._unbindEvents();
    }

    _bindEvents() {
        this._onDragEnter = this._handleDragEnter.bind(this);
        this._onDragOver = this._handleDragOver.bind(this);
        this._onDragLeave = this._handleDragLeave.bind(this);
        this._onDrop = this._handleDrop.bind(this);

        this.el.addEventListener('dragenter', this._onDragEnter);
        this.el.addEventListener('dragover', this._onDragOver);
        this.el.addEventListener('dragleave', this._onDragLeave);
        this.el.addEventListener('drop', this._onDrop);
    }

    _unbindEvents() {
        this.el.removeEventListener('dragenter', this._onDragEnter);
        this.el.removeEventListener('dragover', this._onDragOver);
        this.el.removeEventListener('dragleave', this._onDragLeave);
        this.el.removeEventListener('drop', this._onDrop);
    }

    _isAccepted(e) {
        if (!this.accept) return true;

        // Check if dragged element matches accept selector
        // For HTML5 drag, we check the source element via dataTransfer
        const types = e.dataTransfer?.types || [];

        // If using custom Vango dragging, check window state
        if (window.__vango_dragging__) {
            const dragged = window.__vango_dragging__;
            return dragged.matches(this.accept);
        }

        // Allow file drops by default if no accept specified
        if (types.includes('Files')) return true;

        return true;
    }

    _handleDragEnter(e) {
        if (!this._isAccepted(e)) return;

        e.preventDefault();
        this.el.classList.add(this.hoverClass);
    }

    _handleDragOver(e) {
        if (!this._isAccepted(e)) return;

        e.preventDefault();
        // Set dropEffect to show valid drop cursor
        e.dataTransfer.dropEffect = 'move';
    }

    _handleDragLeave(e) {
        // Only remove class if leaving the element entirely
        if (!this.el.contains(e.relatedTarget)) {
            this.el.classList.remove(this.hoverClass);
        }
    }

    _handleDrop(e) {
        e.preventDefault();
        this.el.classList.remove(this.hoverClass);

        if (!this._isAccepted(e)) return;

        // Gather drop data
        const dropData = {
            x: e.clientX,
            y: e.clientY,
        };

        // Check for custom Vango dragging first
        if (window.__vango_dragging__) {
            const dragged = window.__vango_dragging__;
            dropData.id = dragged.dataset.id || dragged.dataset.hid;
            dropData.sourceHid = dragged.dataset.hid;
            window.__vango_dragging__ = null;
        }

        // Check for files
        if (e.dataTransfer?.files?.length > 0) {
            dropData.fileCount = e.dataTransfer.files.length;
            dropData.fileNames = Array.from(e.dataTransfer.files).map(f => f.name);
        }

        // Check for text/plain data
        const text = e.dataTransfer?.getData('text/plain');
        if (text) {
            dropData.text = text;
        }

        // Check for custom data
        const customData = e.dataTransfer?.getData('application/json');
        if (customData) {
            try {
                dropData.custom = JSON.parse(customData);
            } catch (err) {
                // Ignore parse errors
            }
        }

        this.pushEvent('drop', dropData);
    }
}
