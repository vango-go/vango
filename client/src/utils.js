/**
 * Vango Client Utility Functions
 */

/**
 * Convert HID string ("h42") to integer (42)
 * @param {string} hid - Hydration ID string
 * @returns {number} The numeric portion
 */
export function hidToInt(hid) {
    return parseInt(hid.slice(1), 10);
}

/**
 * Convert integer to HID string
 * @param {number} n - Integer value
 * @returns {string} HID string like "h42"
 */
export function intToHid(n) {
    return 'h' + n;
}

/**
 * Concatenate multiple Uint8Arrays into one
 * @param {Uint8Array[]} arrays - Arrays to concatenate
 * @returns {Uint8Array} Combined array
 */
export function concat(arrays) {
    let totalLength = 0;
    for (let i = 0; i < arrays.length; i++) {
        totalLength += arrays[i].length;
    }

    const result = new Uint8Array(totalLength);
    let offset = 0;

    for (let i = 0; i < arrays.length; i++) {
        result.set(arrays[i], offset);
        offset += arrays[i].length;
    }

    return result;
}

/**
 * Simple debounce function
 * @param {Function} fn - Function to debounce
 * @param {number} delay - Delay in milliseconds
 * @returns {Function} Debounced function
 */
export function debounce(fn, delay) {
    let timer = null;
    return function (...args) {
        if (timer) clearTimeout(timer);
        timer = setTimeout(() => {
            timer = null;
            fn.apply(this, args);
        }, delay);
    };
}

/**
 * Simple throttle function
 * @param {Function} fn - Function to throttle
 * @param {number} limit - Minimum time between calls
 * @returns {Function} Throttled function
 */
export function throttle(fn, limit) {
    let inThrottle = false;
    return function (...args) {
        if (!inThrottle) {
            fn.apply(this, args);
            inThrottle = true;
            setTimeout(() => {
                inThrottle = false;
            }, limit);
        }
    };
}
