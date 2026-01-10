import { ThemeToggleHook } from '../src/hooks/theme_toggle.js';

function click(el) {
    el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
}

describe('ThemeToggleHook', () => {
    let originalMatchMedia;

    beforeAll(() => {
        originalMatchMedia = window.matchMedia;
        window.matchMedia = () => ({ matches: false });
    });

    afterAll(() => {
        window.matchMedia = originalMatchMedia;
    });

    beforeEach(() => {
        document.documentElement.classList.remove('dark');
        localStorage.clear();
    });

    test('toggles documentElement.dark and persists to localStorage', () => {
        const el = document.createElement('button');
        document.body.appendChild(el);

        const hook = new ThemeToggleHook();
        hook.mounted(el, { storageKey: 'theme' });

        expect(document.documentElement.classList.contains('dark')).toBe(false);
        click(el);
        expect(document.documentElement.classList.contains('dark')).toBe(true);
        expect(localStorage.getItem('theme')).toBe('dark');

        click(el);
        expect(document.documentElement.classList.contains('dark')).toBe(false);
        expect(localStorage.getItem('theme')).toBe('light');

        hook.destroyed(el);
        document.body.removeChild(el);
    });

    test('removes click listener on destroyed', () => {
        const el = document.createElement('button');
        document.body.appendChild(el);

        const hook = new ThemeToggleHook();
        hook.mounted(el, { storageKey: 'theme' });

        hook.destroyed(el);

        click(el);
        expect(document.documentElement.classList.contains('dark')).toBe(false);
        expect(localStorage.getItem('theme')).toBe(null);

        document.body.removeChild(el);
    });
});
