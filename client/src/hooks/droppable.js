/**
 * Droppable Hook - Drop Zones
 *
 * Provides drop zone functionality. Fires 'drop' event when element is dropped.
 * Config: accept (CSS selector), hoverClass (default: 'drag-over')
 */

export class DroppableHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;
        this.hoverClass = config.hoverClass || 'drag-over';
        this._bind();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
        this.hoverClass = config.hoverClass || 'drag-over';
    }

    destroyed() { this._unbind(); }

    _bind() {
        this._enter = e => { e.preventDefault(); this.el.classList.add(this.hoverClass); };
        this._over = e => { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; };
        this._leave = e => { if (!this.el.contains(e.relatedTarget)) this.el.classList.remove(this.hoverClass); };
        this._drop = e => {
            e.preventDefault();
            this.el.classList.remove(this.hoverClass);
            const data = { x: e.clientX, y: e.clientY };
            if (window.__vango_dragging__) {
                const d = window.__vango_dragging__;
                data.id = d.dataset.id || d.dataset.hid;
                window.__vango_dragging__ = null;
            }
            const files = e.dataTransfer?.files;
            if (files?.length) data.fileNames = Array.from(files).map(f => f.name);
            const text = e.dataTransfer?.getData('text/plain');
            if (text) data.text = text;
            this.pushEvent('drop', data);
        };
        this.el.addEventListener('dragenter', this._enter);
        this.el.addEventListener('dragover', this._over);
        this.el.addEventListener('dragleave', this._leave);
        this.el.addEventListener('drop', this._drop);
    }

    _unbind() {
        this.el.removeEventListener('dragenter', this._enter);
        this.el.removeEventListener('dragover', this._over);
        this.el.removeEventListener('dragleave', this._leave);
        this.el.removeEventListener('drop', this._drop);
    }
}
