/**
 * Client Hook Lifecycle Management
 *
 * Manages hooks - JavaScript behavior attached to elements that need
 * to run on the client (like sorting, tooltips, dropdowns).
 */

import { SortableHook } from './sortable.js';
import { DraggableHook } from './draggable.js';
import { DroppableHook } from './droppable.js';
import { ResizableHook } from './resizable.js';
import { TooltipHook } from './tooltip.js';
import { DropdownHook } from './dropdown.js';
import { CollapsibleHook } from './collapsible.js';
import { FocusTrapHook } from './focustrap.js';
import { PortalHook } from './portal.js';
import { DialogHook } from './dialog.js';
import { PopoverHook } from './popover.js';
import { ThemeToggleHook } from './theme_toggle.js';

export class HookManager {
    constructor(client) {
        this.client = client;
        this.instances = new Map(); // hid -> { hook, instance, el }
        this.instancesByHookID = new Map(); // vangoHookId -> { hook, instance, el }
        this.pendingReverts = new Map(); // hid -> revertFn
        this._hookIdCounter = 0;

        // Register standard hooks (Section 8.4 of spec)
        this.hooks = {
            'Sortable': SortableHook,
            'Draggable': DraggableHook,
            'Droppable': DroppableHook,
            'Resizable': ResizableHook,
            'Tooltip': TooltipHook,
            'Dropdown': DropdownHook,
            'Collapsible': CollapsibleHook,
            // VangoUI helper hooks
            'FocusTrap': FocusTrapHook,
            'Portal': PortalHook,
            'Dialog': DialogHook,
            'Popover': PopoverHook,
            // App shell helpers
            'ThemeToggle': ThemeToggleHook,
        };

        // Listen for revert events from server
        document.addEventListener('vango:hook-revert', (e) => {
            const hid = e.detail?.hid;
            if (hid && this.pendingReverts.has(hid)) {
                const revertFn = this.pendingReverts.get(hid);
                if (typeof revertFn === 'function') {
                    revertFn();
                }
                this.pendingReverts.delete(hid);
            }
        });
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
            if (hid) {
                if (!this.instances.has(hid)) {
                    this.initializeForNode(el);
                }
                return;
            }

            // Some client-only hooks don't require a server HID. Support these
            // by generating an internal ID stored on the element.
            const hookID = el.dataset.vangoHookId;
            if (!hookID || !this.instancesByHookID.has(hookID)) {
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
        let keyKind = 'hid';
        let key = hid;

        if (!key) {
            keyKind = 'hookID';
            key = el.dataset.vangoHookId;
            if (!key) {
                this._hookIdCounter++;
                key = `hk${this._hookIdCounter}`;
                el.dataset.vangoHookId = key;
            }

            if (this.instancesByHookID.has(key)) {
                return;
            }
        } else {
            // Already initialized
            if (this.instances.has(key)) {
                return;
            }
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

        // Create push event function with optional revert callback
        const pushEvent = (eventName, data = {}, revertFn = null) => {
            if (keyKind !== 'hid') {
                if (this.client.options.debug) {
                    console.warn('[Vango] Hook event ignored: hook element has no data-hid');
                }
                return;
            }
            // Store revert callback if provided
            if (typeof revertFn === 'function') {
                this.pendingReverts.set(key, revertFn);
            }
            this.client.sendHookEvent(key, eventName, data);
        };

        // Instantiate and mount
        const instance = new HookClass();
        instance.mounted(el, config, pushEvent);

        if (keyKind === 'hid') {
            this.instances.set(key, { hook: HookClass, instance, el });
        } else {
            this.instancesByHookID.set(key, { hook: HookClass, instance, el });
        }
    }

    /**
     * Destroy hook for a specific node
     */
    destroyForNode(el) {
        const hid = el.dataset?.hid;
        if (hid) {
            const entry = this.instances.get(hid);
            if (entry) {
                if (entry.instance.destroyed) {
                    entry.instance.destroyed(entry.el);
                }
                this.instances.delete(hid);
            }
            return;
        }

        const hookID = el.dataset?.vangoHookId;
        if (!hookID) return;
        const entry = this.instancesByHookID.get(hookID);
        if (!entry) return;
        if (entry.instance.destroyed) {
            entry.instance.destroyed(entry.el);
        }
        this.instancesByHookID.delete(hookID);
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

        for (const [hookID, entry] of this.instancesByHookID) {
            if (entry.instance.destroyed) {
                entry.instance.destroyed(entry.el);
            }
        }
        this.instancesByHookID.clear();
    }

    /**
     * Update hook config
     */
    updateConfig(hid, config) {
        const entry = this.instances.get(hid);
        if (entry && entry.instance.updated) {
            const pushEvent = (eventName, data = {}, revertFn = null) => {
                if (typeof revertFn === 'function') {
                    this.pendingReverts.set(hid, revertFn);
                }
                this.client.sendHookEvent(hid, eventName, data);
            };
            entry.instance.updated(entry.el, config, pushEvent);
        }
    }
}
