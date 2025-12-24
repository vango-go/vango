/**
 * Preference Sync
 *
 * Client-side preference management with cross-tab sync via BroadcastChannel.
 * Works with both anonymous users (LocalStorage) and authenticated users (server sync).
 */

/**
 * Merge strategies for conflict resolution
 */
export const MergeStrategy = {
    DB_WINS: 0,      // Server value wins
    LOCAL_WINS: 1,   // Local value wins
    PROMPT: 2,       // Ask user to choose
    LWW: 3,          // Last-write-wins by timestamp
};

/**
 * PrefManager handles preference persistence and cross-tab sync.
 */
export class PrefManager {
    constructor(client, options = {}) {
        this.client = client;
        this.options = {
            channelName: 'vango:prefs',
            storagePrefix: 'vango_pref_',
            debug: options.debug || false,
            ...options,
        };

        // Registered preferences
        this.prefs = new Map();

        // BroadcastChannel for cross-tab sync
        this.channel = null;
        if (typeof BroadcastChannel !== 'undefined') {
            this.channel = new BroadcastChannel(this.options.channelName);
            this.channel.onmessage = (event) => this._handleBroadcast(event);
        }

        // Listen for storage events (fallback for browsers without BroadcastChannel)
        if (typeof window !== 'undefined') {
            window.addEventListener('storage', (event) => this._handleStorageEvent(event));
        }
    }

    /**
     * Register a preference
     * @param {string} key - Unique preference key
     * @param {*} defaultValue - Default value
     * @param {Object} options - Configuration options
     * @returns {Pref} Preference instance
     */
    register(key, defaultValue, options = {}) {
        const pref = new Pref(this, key, defaultValue, options);
        this.prefs.set(key, pref);

        // Load from storage
        pref._loadFromStorage();

        return pref;
    }

    /**
     * Get a registered preference
     */
    get(key) {
        return this.prefs.get(key);
    }

    /**
     * Broadcast a preference change to other tabs
     */
    broadcast(key, value, updatedAt) {
        if (this.channel) {
            this.channel.postMessage({
                type: 'update',
                key,
                value,
                updatedAt: updatedAt.toISOString(),
            });
        }
    }

    /**
     * Handle broadcast message from another tab
     */
    _handleBroadcast(event) {
        const { type, key, value, updatedAt } = event.data;

        if (type === 'update') {
            const pref = this.prefs.get(key);
            if (pref) {
                pref._setFromRemote(value, new Date(updatedAt));
            }
        }

        if (this.options.debug) {
            console.log('[Vango Prefs] Broadcast received:', event.data);
        }
    }

    /**
     * Handle storage event (fallback for cross-tab sync)
     */
    _handleStorageEvent(event) {
        if (!event.key || !event.key.startsWith(this.options.storagePrefix)) {
            return;
        }

        const prefKey = event.key.slice(this.options.storagePrefix.length);
        const pref = this.prefs.get(prefKey);

        if (pref && event.newValue) {
            try {
                const data = JSON.parse(event.newValue);
                pref._setFromRemote(data.value, new Date(data.updatedAt));
            } catch (e) {
                if (this.options.debug) {
                    console.warn('[Vango Prefs] Failed to parse storage event:', e);
                }
            }
        }
    }

    /**
     * Save to LocalStorage
     */
    saveToStorage(key, value, updatedAt) {
        if (typeof localStorage === 'undefined') return;

        const storageKey = this.options.storagePrefix + key;
        const data = {
            value,
            updatedAt: updatedAt.toISOString(),
        };

        try {
            localStorage.setItem(storageKey, JSON.stringify(data));
        } catch (e) {
            if (this.options.debug) {
                console.warn('[Vango Prefs] Failed to save to storage:', e);
            }
        }
    }

    /**
     * Load from LocalStorage
     */
    loadFromStorage(key) {
        if (typeof localStorage === 'undefined') return null;

        const storageKey = this.options.storagePrefix + key;
        try {
            const data = localStorage.getItem(storageKey);
            if (data) {
                const parsed = JSON.parse(data);
                return {
                    value: parsed.value,
                    updatedAt: new Date(parsed.updatedAt),
                };
            }
        } catch (e) {
            if (this.options.debug) {
                console.warn('[Vango Prefs] Failed to load from storage:', e);
            }
        }
        return null;
    }

    /**
     * Remove from LocalStorage
     */
    removeFromStorage(key) {
        if (typeof localStorage === 'undefined') return;

        const storageKey = this.options.storagePrefix + key;
        try {
            localStorage.removeItem(storageKey);
        } catch (e) {
            // Ignore errors
        }
    }

    /**
     * Sync all preferences with server
     * Called when user logs in
     */
    async syncWithServer(serverPrefs) {
        for (const [key, serverData] of Object.entries(serverPrefs)) {
            const pref = this.prefs.get(key);
            if (pref) {
                pref._mergeWithServer(serverData.value, new Date(serverData.updatedAt));
            }
        }
    }

    /**
     * Get all preferences as an object for server sync
     */
    getAllForSync() {
        const result = {};
        for (const [key, pref] of this.prefs) {
            if (pref.options.syncToServer !== false) {
                result[key] = {
                    value: pref.value,
                    updatedAt: pref.updatedAt.toISOString(),
                };
            }
        }
        return result;
    }

    /**
     * Cleanup
     */
    destroy() {
        if (this.channel) {
            this.channel.close();
            this.channel = null;
        }
    }
}

/**
 * Individual preference instance
 */
export class Pref {
    constructor(manager, key, defaultValue, options = {}) {
        this.manager = manager;
        this.key = key;
        this.defaultValue = defaultValue;
        this.value = defaultValue;
        this.updatedAt = new Date();

        this.options = {
            mergeStrategy: MergeStrategy.LWW,
            persistLocal: true,
            syncToServer: true,
            onConflict: null,  // Custom conflict handler
            onChange: null,    // Called when value changes
            ...options,
        };

        // Subscribers for reactive updates
        this.subscribers = new Set();
    }

    /**
     * Get the current value
     */
    get() {
        return this.value;
    }

    /**
     * Set a new value
     */
    set(value) {
        if (this._isEqual(this.value, value)) {
            return;
        }

        const oldValue = this.value;
        this.value = value;
        this.updatedAt = new Date();

        // Persist locally
        if (this.options.persistLocal) {
            this.manager.saveToStorage(this.key, value, this.updatedAt);
        }

        // Broadcast to other tabs
        this.manager.broadcast(this.key, value, this.updatedAt);

        // Sync to server via client
        if (this.options.syncToServer && this.manager.client) {
            this._syncToServer();
        }

        // Notify subscribers
        this._notifySubscribers(value, oldValue);

        // Call onChange callback
        if (this.options.onChange) {
            this.options.onChange(value, oldValue);
        }
    }

    /**
     * Reset to default value
     */
    reset() {
        this.set(this.defaultValue);
    }

    /**
     * Subscribe to value changes
     * @param {Function} callback - Called with (newValue, oldValue)
     * @returns {Function} Unsubscribe function
     */
    subscribe(callback) {
        this.subscribers.add(callback);
        return () => this.subscribers.delete(callback);
    }

    /**
     * Load initial value from storage
     */
    _loadFromStorage() {
        const stored = this.manager.loadFromStorage(this.key);
        if (stored) {
            this.value = stored.value;
            this.updatedAt = stored.updatedAt;
        }
    }

    /**
     * Set value from remote source (another tab)
     */
    _setFromRemote(value, remoteUpdatedAt) {
        const resolved = this._resolveConflict(this.value, value, this.updatedAt, remoteUpdatedAt);

        if (!this._isEqual(this.value, resolved)) {
            const oldValue = this.value;
            this.value = resolved;

            // Update timestamp if remote was newer
            if (remoteUpdatedAt > this.updatedAt) {
                this.updatedAt = remoteUpdatedAt;
            }

            // Update storage
            if (this.options.persistLocal) {
                this.manager.saveToStorage(this.key, this.value, this.updatedAt);
            }

            // Notify subscribers
            this._notifySubscribers(this.value, oldValue);

            if (this.options.onChange) {
                this.options.onChange(this.value, oldValue);
            }
        }
    }

    /**
     * Merge with server value (called on login)
     */
    _mergeWithServer(serverValue, serverUpdatedAt) {
        const resolved = this._resolveConflict(this.value, serverValue, this.updatedAt, serverUpdatedAt);

        if (!this._isEqual(this.value, resolved)) {
            const oldValue = this.value;
            this.value = resolved;

            // Use the more recent timestamp
            if (serverUpdatedAt > this.updatedAt) {
                this.updatedAt = serverUpdatedAt;
            }

            // Update storage
            if (this.options.persistLocal) {
                this.manager.saveToStorage(this.key, this.value, this.updatedAt);
            }

            // Notify subscribers
            this._notifySubscribers(this.value, oldValue);

            if (this.options.onChange) {
                this.options.onChange(this.value, oldValue);
            }
        }
    }

    /**
     * Resolve conflict between local and remote values
     */
    _resolveConflict(local, remote, localTime, remoteTime) {
        // Custom handler takes precedence
        if (this.options.onConflict) {
            return this.options.onConflict(local, remote, localTime, remoteTime);
        }

        switch (this.options.mergeStrategy) {
            case MergeStrategy.DB_WINS:
                return remote;

            case MergeStrategy.LOCAL_WINS:
                return local;

            case MergeStrategy.LWW:
                return remoteTime > localTime ? remote : local;

            case MergeStrategy.PROMPT:
                // For PROMPT, we'd need to show UI to user
                // For now, fall back to LWW
                return remoteTime > localTime ? remote : local;

            default:
                return local;
        }
    }

    /**
     * Sync preference to server
     */
    _syncToServer() {
        if (!this.manager.client || !this.manager.client.connected) {
            return;
        }

        // Send preference update event
        this.manager.client.sendEvent(
            0x10, // EventType.PREF (custom event type for preferences)
            '',   // No HID needed
            {
                key: this.key,
                value: this.value,
                updatedAt: this.updatedAt.toISOString(),
            }
        );
    }

    /**
     * Notify all subscribers of value change
     */
    _notifySubscribers(newValue, oldValue) {
        for (const callback of this.subscribers) {
            try {
                callback(newValue, oldValue);
            } catch (e) {
                console.error('[Vango Prefs] Subscriber error:', e);
            }
        }
    }

    /**
     * Check if two values are equal
     */
    _isEqual(a, b) {
        if (a === b) return true;
        if (typeof a !== typeof b) return false;
        if (typeof a === 'object' && a !== null && b !== null) {
            return JSON.stringify(a) === JSON.stringify(b);
        }
        return false;
    }
}

/**
 * Default export for convenience
 */
export default PrefManager;
