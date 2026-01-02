/**
 * Resizable Hook - Resize Handles
 *
 * Provides resize handles on elements. Fires 'resize' event when resize completes.
 *
 * Config: handles (default: 'se'), minWidth, maxWidth, minHeight, maxHeight
 */

export class ResizableHook {
    mounted(el, config, pushEvent) {
        this.el = el;
        this.config = config;
        this.pushEvent = pushEvent;
        this.handles = (config.handles || 'se').split(',').map(h => h.trim());
        this.resizing = false;
        this._handleEls = [];
        this._createHandles();
        this._bind();
    }

    updated(el, config, pushEvent) {
        this.config = config;
        this.pushEvent = pushEvent;
    }

    destroyed() {
        this._unbind();
        this._handleEls.forEach(h => h.remove());
    }

    _createHandles() {
        if (getComputedStyle(this.el).position === 'static') {
            this.el.style.position = 'relative';
        }
        const cursors = { n: 'n', s: 's', e: 'e', w: 'w', ne: 'ne', se: 'se', sw: 'sw', nw: 'nw' };
        for (const h of this.handles) {
            if (!cursors[h]) continue;
            const el = document.createElement('div');
            el.dataset.handle = h;
            el.style.cssText = `position:absolute;z-index:10;cursor:${h}-resize;` + this._pos(h);
            this.el.appendChild(el);
            this._handleEls.push(el);
        }
    }

    _pos(h) {
        const s = 8, hs = 4;
        const m = { n: `top:-${hs}px;left:25%;width:50%;height:${s}px`,
            s: `bottom:-${hs}px;left:25%;width:50%;height:${s}px`,
            e: `right:-${hs}px;top:25%;width:${s}px;height:50%`,
            w: `left:-${hs}px;top:25%;width:${s}px;height:50%`,
            ne: `top:-${hs}px;right:-${hs}px;width:${s}px;height:${s}px`,
            se: `bottom:-${hs}px;right:-${hs}px;width:${s}px;height:${s}px`,
            sw: `bottom:-${hs}px;left:-${hs}px;width:${s}px;height:${s}px`,
            nw: `top:-${hs}px;left:-${hs}px;width:${s}px;height:${s}px` };
        return m[h] || '';
    }

    _bind() {
        this._onDown = e => { e.preventDefault(); this._start(e.target.dataset.handle, e.clientX, e.clientY); };
        this._onMove = e => { if (this.resizing) { e.preventDefault(); this._update(e.clientX, e.clientY); } };
        this._onUp = () => { if (this.resizing) this._end(); };
        this._handleEls.forEach(h => h.addEventListener('mousedown', this._onDown));
        document.addEventListener('mousemove', this._onMove);
        document.addEventListener('mouseup', this._onUp);
    }

    _unbind() {
        this._handleEls.forEach(h => h.removeEventListener('mousedown', this._onDown));
        document.removeEventListener('mousemove', this._onMove);
        document.removeEventListener('mouseup', this._onUp);
    }

    _start(handle, x, y) {
        this.resizing = true;
        this._h = handle;
        this._sx = x; this._sy = y;
        const r = this.el.getBoundingClientRect();
        this._sw = r.width; this._sh = r.height;
    }

    _update(x, y) {
        const dx = x - this._sx, dy = y - this._sy;
        let w = this._sw, h = this._sh;
        if (this._h.includes('e')) w = this._sw + dx;
        if (this._h.includes('w')) w = this._sw - dx;
        if (this._h.includes('s')) h = this._sh + dy;
        if (this._h.includes('n')) h = this._sh - dy;
        const c = this.config;
        w = Math.max(c.minWidth || 0, Math.min(c.maxWidth || Infinity, w));
        h = Math.max(c.minHeight || 0, Math.min(c.maxHeight || Infinity, h));
        this.el.style.width = w + 'px';
        this.el.style.height = h + 'px';
    }

    _end() {
        this.resizing = false;
        const r = this.el.getBoundingClientRect();
        this.pushEvent('resize', { width: Math.round(r.width), height: Math.round(r.height) });
    }
}
