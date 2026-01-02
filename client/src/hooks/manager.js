/**
 * Client Hook Lifecycle Management
 *
 * Manages hooks - JavaScript behavior attached to elements that need
 * to run on the client (like sorting, tooltips, dropdowns).
 */

import { SortableHook } from './sortable.js';
import { DraggableHook } from './draggable.js';
import { TooltipHook } from './tooltip.js';
import { DropdownHook } from './dropdown.js';
import { FocusTrapHook } from './focustrap.js';
import { PortalHook } from './portal.js';
import { DialogHook } from './dialog.js';
import { PopoverHook } from './popover.js';

export class HookManager {
    constructor(client) {
        this.client = client;
        this.instances = new Map(); // hid -> { hook, instance, el }

        // Register standard hooks
        this.hooks = {
            'Sortable': SortableHook,
            'Draggable': DraggableHook,
            'Tooltip': TooltipHook,
            'Dropdown': DropdownHook,
            'FocusTrap': FocusTrapHook,
            'Portal': PortalHook,
            'Dialog': DialogHook,
            'Popover': PopoverHook,
        };
    }

    /**
     * Register a custom hook
     */
    register(name, hookClass) {
        this.hooks[name] = hookClass;
    }

    /**
     * Initialize hooks from current DOM
     */
    initializeFromDOM() {
        document.querySelectorAll('[data-hook]').forEach(el => {
            this.initializeForNode(el);
        });
    }

    /**
     * Update hooks after DOM changes
     */
    updateFromDOM() {
        // Initialize any new hooks
        document.querySelectorAll('[data-hook]').forEach(el => {
            const hid = el.dataset.hid;
            if (hid && !this.instances.has(hid)) {
                this.initializeForNode(el);
            }
        });
    }

    /**
     * Initialize hook for a specific node
     */
    initializeForNode(el) {
        const hookName = el.dataset.hook;
        if (!hookName) return;

        const HookClass = this.hooks[hookName];
        if (!HookClass) {
            if (this.client.options.debug) {
                console.warn('[Vango] Unknown hook:', hookName);
            }
            return;
        }

        const hid = el.dataset.hid;
        if (!hid) {
            if (this.client.options.debug) {
                console.warn('[Vango] Hook element must have data-hid');
            }
            return;
        }

        // Already initialized
        if (this.instances.has(hid)) {
            return;
        }

        // Parse config from data-hook-config
        let config = {};
        try {
            if (el.dataset.hookConfig) {
                config = JSON.parse(el.dataset.hookConfig);
            }
        } catch (e) {
            if (this.client.options.debug) {
                console.warn('[Vango] Invalid hook config:', e);
            }
        }

        // Create push event function
        const pushEvent = (eventName, data = {}) => {
            this.client.sendHookEvent(hid, eventName, data);
        };

        // Instantiate and mount
        const instance = new HookClass();
        instance.mounted(el, config, pushEvent);

        this.instances.set(hid, { hook: HookClass, instance, el });
    }

    /**
     * Destroy hook for a specific node
     */
    destroyForNode(el) {
        const hid = el.dataset?.hid;
        if (!hid) return;

        const entry = this.instances.get(hid);
        if (entry) {
            if (entry.instance.destroyed) {
                entry.instance.destroyed(entry.el);
            }
            this.instances.delete(hid);
        }
    }

    /**
     * Destroy all hooks
     */
    destroyAll() {
        for (const [hid, entry] of this.instances) {
            if (entry.instance.destroyed) {
                entry.instance.destroyed(entry.el);
            }
        }
        this.instances.clear();
    }

    /**
     * Update hook config
     */
    updateConfig(hid, config) {
        const entry = this.instances.get(hid);
        if (entry && entry.instance.updated) {
            const pushEvent = (eventName, data = {}) => {
                this.client.sendHookEvent(hid, eventName, data);
            };
            entry.instance.updated(entry.el, config, pushEvent);
        }
    }
}
