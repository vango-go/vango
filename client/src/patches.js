/**
 * Patch Application
 *
 * Applies DOM patches received from the server.
 */

import { PatchType } from './codec.js';

export class PatchApplier {
    constructor(client) {
        this.client = client;
    }

    /**
     * Apply array of patches
     */
    apply(patches) {
        for (const patch of patches) {
            this.applyPatch(patch);
        }
    }

    /**
     * Apply single patch
     */
    applyPatch(patch) {
        const el = this.client.getNode(patch.hid);

        // Some patches don't require the target element to exist
        if (!el && patch.type !== PatchType.INSERT_NODE) {
            if (this.client.options.debug) {
                console.warn('[Vango] Node not found:', patch.hid);
            }
            return;
        }

        switch (patch.type) {
            case PatchType.SET_TEXT:
                this._setText(el, patch.value);
                break;

            case PatchType.SET_ATTR:
                this._setAttr(el, patch.key, patch.value);
                break;

            case PatchType.REMOVE_ATTR:
                this._removeAttr(el, patch.key);
                break;

            case PatchType.ADD_CLASS:
                el.classList.add(patch.className);
                break;

            case PatchType.REMOVE_CLASS:
                el.classList.remove(patch.className);
                break;

            case PatchType.TOGGLE_CLASS:
                el.classList.toggle(patch.className);
                break;

            case PatchType.SET_STYLE:
                el.style[patch.key] = patch.value;
                break;

            case PatchType.REMOVE_STYLE:
                el.style[patch.key] = '';
                break;

            case PatchType.INSERT_NODE:
                this._insertNode(patch.parentID, patch.index, patch.vnode);
                break;

            case PatchType.REMOVE_NODE:
                this._removeNode(el, patch.hid);
                break;

            case PatchType.MOVE_NODE:
                this._moveNode(el, patch.parentID, patch.index);
                break;

            case PatchType.REPLACE_NODE:
                this._replaceNode(el, patch.hid, patch.vnode);
                break;

            case PatchType.SET_VALUE:
                this._setValue(el, patch.value);
                break;

            case PatchType.SET_CHECKED:
                el.checked = patch.value;
                break;

            case PatchType.SET_SELECTED:
                el.selected = patch.value;
                break;

            case PatchType.FOCUS:
                el.focus();
                break;

            case PatchType.BLUR:
                el.blur();
                break;

            case PatchType.SCROLL_TO:
                el.scrollTo({
                    left: patch.x,
                    top: patch.y,
                    behavior: patch.behavior === 1 ? 'smooth' : 'instant',
                });
                break;

            case PatchType.SET_DATA:
                el.dataset[patch.key] = patch.value;
                break;

            case PatchType.DISPATCH:
                this._dispatchEvent(el, patch.eventName, patch.detail);
                break;

            case PatchType.EVAL:
                this._evalCode(patch.code);
                break;

            default:
                if (this.client.options.debug) {
                    console.warn('[Vango] Unknown patch type:', patch.type);
                }
        }
    }

    /**
     * Set text content
     */
    _setText(el, text) {
        el.textContent = text;
    }

    /**
     * Set attribute with special cases
     */
    _setAttr(el, key, value) {
        switch (key) {
            case 'class':
                el.className = value;
                break;
            case 'for':
                el.htmlFor = value;
                break;
            case 'value':
                this._setValue(el, value);
                break;
            case 'checked':
                el.checked = value === 'true' || value === '';
                break;
            case 'selected':
                el.selected = value === 'true' || value === '';
                break;
            case 'disabled':
            case 'readonly':
            case 'required':
            case 'multiple':
            case 'autofocus':
                // Boolean attributes
                if (value === 'true' || value === '') {
                    el.setAttribute(key, '');
                } else {
                    el.removeAttribute(key);
                }
                break;
            default:
                el.setAttribute(key, value);
        }
    }

    /**
     * Remove attribute
     */
    _removeAttr(el, key) {
        switch (key) {
            case 'class':
                el.className = '';
                break;
            case 'for':
                el.htmlFor = '';
                break;
            case 'value':
                el.value = '';
                break;
            case 'checked':
                el.checked = false;
                break;
            case 'selected':
                el.selected = false;
                break;
            default:
                el.removeAttribute(key);
        }
    }

    /**
     * Set input value (preserving cursor position)
     */
    _setValue(el, value) {
        if (el.value === value) return;

        // Preserve cursor position for text inputs
        if (el.type === 'text' || el.tagName === 'TEXTAREA') {
            const start = el.selectionStart;
            const end = el.selectionEnd;
            el.value = value;

            // Restore cursor if element is focused
            if (document.activeElement === el) {
                el.setSelectionRange(
                    Math.min(start, value.length),
                    Math.min(end, value.length)
                );
            }
        } else {
            el.value = value;
        }
    }

    /**
     * Insert node at index in parent
     */
    _insertNode(parentHid, index, vnode) {
        const parentEl = this.client.getNode(parentHid);
        if (!parentEl) {
            if (this.client.options.debug) {
                console.warn('[Vango] Parent node not found:', parentHid);
            }
            return;
        }

        const newEl = this._createNode(vnode);

        if (index >= parentEl.children.length) {
            parentEl.appendChild(newEl);
        } else {
            parentEl.insertBefore(newEl, parentEl.children[index]);
        }
    }

    /**
     * Remove node
     */
    _removeNode(el, hid) {
        // Cleanup hooks
        this.client.hooks.destroyForNode(el);

        // Remove from map
        this.client.unregisterNode(hid);

        // Unregister all children with HIDs
        el.querySelectorAll('[data-hid]').forEach(child => {
            this.client.hooks.destroyForNode(child);
            this.client.unregisterNode(child.dataset.hid);
        });

        // Remove from DOM
        el.remove();
    }

    /**
     * Move node to new position
     */
    _moveNode(el, parentHid, index) {
        const parentEl = this.client.getNode(parentHid);
        if (!parentEl) {
            if (this.client.options.debug) {
                console.warn('[Vango] Parent node not found:', parentHid);
            }
            return;
        }

        if (index >= parentEl.children.length) {
            parentEl.appendChild(el);
        } else {
            parentEl.insertBefore(el, parentEl.children[index]);
        }
    }

    /**
     * Replace node
     */
    _replaceNode(el, hid, vnode) {
        // Cleanup old
        this.client.hooks.destroyForNode(el);
        this.client.unregisterNode(hid);
        el.querySelectorAll('[data-hid]').forEach(child => {
            this.client.hooks.destroyForNode(child);
            this.client.unregisterNode(child.dataset.hid);
        });

        // Create and insert new
        const newEl = this._createNode(vnode);
        el.replaceWith(newEl);
    }

    /**
     * Create DOM node from VNode
     */
    _createNode(vnode) {
        switch (vnode.type) {
            case 'element':
                return this._createElement(vnode);
            case 'text':
                return document.createTextNode(vnode.text);
            case 'fragment':
                return this._createFragment(vnode);
            default:
                if (this.client.options.debug) {
                    console.warn('[Vango] Unknown vnode type:', vnode.type);
                }
                return document.createTextNode('');
        }
    }

    /**
     * Create element from VNode
     */
    _createElement(vnode) {
        const el = document.createElement(vnode.tag);

        // Set attributes
        for (const [key, value] of Object.entries(vnode.attrs || {})) {
            this._setAttr(el, key, value);
        }

        // Set HID and register
        if (vnode.hid) {
            el.dataset.hid = vnode.hid;
            this.client.registerNode(vnode.hid, el);
        }

        // Create children
        for (const child of vnode.children || []) {
            el.appendChild(this._createNode(child));
        }

        // Initialize hooks
        this.client.hooks.initializeForNode(el);

        return el;
    }

    /**
     * Create document fragment
     */
    _createFragment(vnode) {
        const frag = document.createDocumentFragment();
        for (const child of vnode.children || []) {
            frag.appendChild(this._createNode(child));
        }
        return frag;
    }

    /**
     * Dispatch custom event
     */
    _dispatchEvent(el, eventName, detail) {
        let parsedDetail = null;
        if (detail) {
            try {
                parsedDetail = JSON.parse(detail);
            } catch (e) {
                parsedDetail = detail;
            }
        }

        const event = new CustomEvent(eventName, {
            detail: parsedDetail,
            bubbles: true,
            cancelable: true,
        });
        el.dispatchEvent(event);
    }

    /**
     * Evaluate JavaScript code (use sparingly!)
     */
    _evalCode(code) {
        try {
            // eslint-disable-next-line no-new-func
            new Function(code)();
        } catch (e) {
            console.error('[Vango] Eval error:', e);
        }
    }
}
