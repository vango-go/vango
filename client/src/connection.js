/**
 * Connection State Manager
 *
 * Manages connection state, CSS classes, and optional toast notifications.
 * Provides visual feedback to users during connection interruptions.
 */

/**
 * Connection states
 */
export const ConnectionState = {
    CONNECTING: 'connecting',
    CONNECTED: 'connected',
    RECONNECTING: 'reconnecting',
    DISCONNECTED: 'disconnected',
};

/**
 * CSS classes applied to document.documentElement (<html>)
 */
const CSS_CLASSES = {
    [ConnectionState.CONNECTING]: 'vango-connecting',
    [ConnectionState.CONNECTED]: 'vango-connected',
    [ConnectionState.RECONNECTING]: 'vango-reconnecting',
    [ConnectionState.DISCONNECTED]: 'vango-disconnected',
};

/**
 * ConnectionManager handles connection state and user feedback.
 */
export class ConnectionManager {
    constructor(options = {}) {
        this.options = {
            // Toast settings
            toastOnReconnect: options.toastOnReconnect || false,
            toastMessage: options.toastMessage || 'Connection restored',
            toastDuration: options.toastDuration || 3000,

            // Reconnection settings
            maxRetries: options.maxRetries || 10,
            baseDelay: options.baseDelay || 1000,
            maxDelay: options.maxDelay || 30000,

            // Debug mode
            debug: options.debug || false,

            ...options,
        };

        this.state = ConnectionState.CONNECTING;
        this.retryCount = 0;
        this.previousState = null;
        this.disconnectReason = null;

        // Apply initial connecting state
        this._updateClasses();
    }

    /**
     * Set connection state and update UI
     */
    setState(newState) {
        if (this.state === newState) return;

        this.previousState = this.state;
        this.state = newState;
        if (newState !== ConnectionState.DISCONNECTED) {
            this.disconnectReason = null;
        }

        // Update body classes
        this._updateClasses();

        // Dispatch custom event
        this._dispatchEvent(newState, this.previousState);

        // Show toast on reconnection
        if (this.options.toastOnReconnect &&
            newState === ConnectionState.CONNECTED &&
            this.previousState !== ConnectionState.CONNECTED) {
            this.showToast(this.options.toastMessage);
        }

        if (this.options.debug) {
            console.log(`[Vango] Connection state: ${this.previousState} -> ${newState}`);
        }
    }

    /**
     * Update CSS classes on document.documentElement (<html>)
     */
    _updateClasses() {
        const root = document.documentElement;

        // Remove all connection state classes
        Object.values(CSS_CLASSES).forEach(cls => {
            root.classList.remove(cls);
        });

        // Add current state class
        const currentClass = CSS_CLASSES[this.state];
        if (currentClass) {
            root.classList.add(currentClass);
        }

        if (this.state === ConnectionState.DISCONNECTED && this.disconnectReason) {
            root.setAttribute('data-vango-disconnect-reason', this.disconnectReason);
        } else {
            root.removeAttribute('data-vango-disconnect-reason');
        }
    }

    /**
     * Dispatch custom event for connection state changes
     */
    _dispatchEvent(state, previousState) {
        const event = new CustomEvent('vango:connection', {
            detail: { state, previousState },
            bubbles: true,
        });
        document.dispatchEvent(event);
    }

    /**
     * Handle connection established
     */
    onConnect() {
        this.retryCount = 0;
        this.setState(ConnectionState.CONNECTED);
    }

    /**
     * Handle connection lost
     */
    onDisconnect() {
        if (this.state === ConnectionState.CONNECTED) {
            this.setState(ConnectionState.RECONNECTING);
        }
    }

    /**
     * Mark the connection as disconnected with an optional reason.
     */
    setDisconnected(reason) {
        this.disconnectReason = reason || null;
        this.setState(ConnectionState.DISCONNECTED);
    }

    /**
     * Handle failed reconnection attempt
     */
    onReconnectFailed() {
        this.retryCount++;

        if (this.retryCount >= this.options.maxRetries) {
            this.setState(ConnectionState.DISCONNECTED);
            return false; // Stop retrying
        }

        return true; // Continue retrying
    }

    /**
     * Calculate next retry delay with exponential backoff
     */
    getNextRetryDelay() {
        const delay = Math.min(
            this.options.baseDelay * Math.pow(2, this.retryCount),
            this.options.maxDelay
        );
        return delay;
    }

    /**
     * Get current state
     */
    getState() {
        return this.state;
    }

    /**
     * Check if currently connected
     */
    isConnected() {
        return this.state === ConnectionState.CONNECTED;
    }

    /**
     * Check if disconnected (gave up reconnecting)
     */
    isDisconnected() {
        return this.state === ConnectionState.DISCONNECTED;
    }

    /**
     * Reset state (e.g., for manual retry)
     */
    reset() {
        this.retryCount = 0;
        this.setState(ConnectionState.CONNECTED);
    }

    /**
     * Show a toast notification
     */
    showToast(message, duration = this.options.toastDuration) {
        // Check if custom toast handler is defined
        if (typeof window.__VANGO_TOAST__ === 'function') {
            window.__VANGO_TOAST__(message, duration);
            return;
        }

        // Create simple toast element
        const toast = document.createElement('div');
        toast.className = 'vango-toast';
        toast.textContent = message;
        toast.setAttribute('role', 'alert');
        toast.setAttribute('aria-live', 'polite');

        // Apply styles
        Object.assign(toast.style, {
            position: 'fixed',
            bottom: '20px',
            left: '50%',
            transform: 'translateX(-50%)',
            padding: '12px 24px',
            backgroundColor: 'hsl(var(--primary, 220 13% 18%))',
            color: 'hsl(var(--primary-foreground, 0 0% 100%))',
            borderRadius: 'var(--radius, 6px)',
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
            zIndex: '10001',
            opacity: '0',
            transition: 'opacity 0.3s ease',
            fontFamily: 'system-ui, -apple-system, sans-serif',
            fontSize: '14px',
        });

        document.body.appendChild(toast);

        // Animate in
        requestAnimationFrame(() => {
            toast.style.opacity = '1';
        });

        // Remove after duration
        setTimeout(() => {
            toast.style.opacity = '0';
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
            }, 300);
        }, duration);
    }
}

/**
 * Inject default reconnection styles
 * These can be overridden by the application's CSS
 */
export function injectDefaultStyles() {
    // Check if styles already injected
    if (document.getElementById('vango-connection-styles')) {
        return;
    }

    const style = document.createElement('style');
    style.id = 'vango-connection-styles';
    style.textContent = `
        /* Vango Connection State Styles */

        /* Connecting indicator - pulsing bar at top */
        html.vango-connecting::after {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            height: 3px;
            background: linear-gradient(90deg, transparent, var(--vango-primary, #3b82f6), transparent);
            animation: vango-connect-pulse 1.5s ease-in-out infinite;
            z-index: 9999;
        }

        @keyframes vango-connect-pulse {
            0%, 100% { opacity: 0.3; }
            50% { opacity: 1; }
        }

        /* Reconnecting indicator - pulsing bar at top */
        html.vango-reconnecting::after {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            height: 3px;
            background: linear-gradient(90deg, transparent, var(--vango-primary, #3b82f6), transparent);
            animation: vango-reconnect-pulse 1.5s ease-in-out infinite;
            z-index: 9999;
        }

        @keyframes vango-reconnect-pulse {
            0%, 100% { opacity: 0.3; }
            50% { opacity: 1; }
        }

        /* Disconnected state - slight grayscale and overlay */
        html.vango-disconnected {
            filter: grayscale(0.3);
        }

        html.vango-disconnected::before {
            content: 'Connection lost. Click to retry...';
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: hsl(var(--background, 0 0% 100%));
            border: 1px solid hsl(var(--border, 0 0% 90%));
            padding: 16px 32px;
            border-radius: var(--radius, 6px);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            z-index: 10000;
            font-family: system-ui, -apple-system, sans-serif;
            font-size: 14px;
            color: hsl(var(--foreground, 0 0% 9%));
            cursor: pointer;
        }

        /* Toast notifications */
        .vango-toast {
            animation: vango-toast-slide-up 0.3s ease-out;
        }

        @keyframes vango-toast-slide-up {
            from {
                transform: translateX(-50%) translateY(20px);
                opacity: 0;
            }
            to {
                transform: translateX(-50%) translateY(0);
                opacity: 1;
            }
        }
    `;

    document.head.appendChild(style);
}

/**
 * Default export for convenience
 */
export default ConnectionManager;
