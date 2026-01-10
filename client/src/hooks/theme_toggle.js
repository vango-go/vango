/**
 * ThemeToggleHook
 *
 * Client-only theme toggle that survives SPA navigation and DOM patching.
 * Toggles the `dark` class on <html> and persists the choice in localStorage.
 *
 * Expected markup:
 *   <button data-hook="ThemeToggle" data-hook-config='{"storageKey":"theme"}'>...</button>
 */

export class ThemeToggleHook {
    mounted(el, config = {}) {
        const storageKey = (config && config.storageKey) ? String(config.storageKey) : 'theme';

        const applyStoredTheme = () => {
            let stored = '';
            try {
                stored = localStorage.getItem(storageKey) || '';
            } catch {
                stored = '';
            }

            const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme:dark)').matches;
            const shouldBeDark = stored === 'dark' || (!stored && prefersDark);

            document.documentElement.classList.toggle('dark', shouldBeDark);
        };

        const toggle = () => {
            const nextIsDark = !document.documentElement.classList.contains('dark');
            document.documentElement.classList.toggle('dark', nextIsDark);
            try {
                localStorage.setItem(storageKey, nextIsDark ? 'dark' : 'light');
            } catch {
                // Ignore storage failures (private mode, quota, etc.)
            }
        };

        // Ensure the current DOM reflects any persisted theme if the element is
        // mounted after a patch/resync cycle.
        applyStoredTheme();

        this._onClick = (e) => {
            // Prevent form submits if embedded in a form by accident.
            e.preventDefault();
            toggle();
        };

        el.addEventListener('click', this._onClick);
    }

    destroyed(el) {
        if (this._onClick) {
            el.removeEventListener('click', this._onClick);
        }
        this._onClick = null;
    }
}
